// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/util/compression"
	"net"
	"net/url"
	"sync/atomic"
	"time"
)

// HandshakeSys struct
type HandshakeSys struct {
	Dict       map[string]uint16 `json:"dict"`
	Heartbeat  int               `json:"heartbeat"`
	Serializer string            `json:"serializer"`
}

// HandshakeResponse struct
type HandshakeResponse struct {
	Code int          `json:"code"`
	Sys  HandshakeSys `json:"sys"`
}

type pendingRequest struct {
	msg    *message.Message
	sentAt time.Time
}

// Client struct
type Client struct {
	conn             net.Conn
	isConnected      int32
	packetEncoder    codec.PacketEncoder
	packetDecoder    codec.PacketDecoder
	packetChan       chan *packet.Packet
	IncomingMsgChan  chan *message.Message
	requestTimeout   time.Duration
	nextId           uint32
	messageEncoder   message.Encoder
	handshakeRequest *HandshakeRequest
	wc               loom.WaitClose
}

// MsgChannel return the incoming message channel
func (c *Client) MsgChannel() chan *message.Message {
	return c.IncomingMsgChan
}

// IsConnected return the connection status
func (c *Client) IsConnected() bool {
	return atomic.LoadInt32(&c.isConnected) == 1
}

// New returns a new client
func New(requestTimeout ...time.Duration) *Client {

	reqTimeout := 5 * time.Second
	if len(requestTimeout) > 0 {
		reqTimeout = requestTimeout[0]
	}

	return &Client{
		isConnected:    0,
		packetEncoder:  codec.NewPomeloPacketEncoder(),
		packetDecoder:  codec.NewPomeloPacketDecoder(),
		packetChan:     make(chan *packet.Packet, 10),
		requestTimeout: reqTimeout,
		messageEncoder: message.NewMessagesEncoder(false),
		handshakeRequest: &HandshakeRequest{
			Sys: HandshakeClientData{
				Platform:    "mac",
				LibVersion:  "0.3.5-release",
				BuildNumber: "20",
				Version:     "2.1",
			},
			User: map[string]interface{}{
				"age": 30,
			},
		},
	}
}

// SetClientHandshakeData sets the data to send inside handshake
func (c *Client) SetHandshakeRequest(data *HandshakeRequest) {
	c.handshakeRequest = data
}

func (c *Client) sendHandshakeRequest() error {
	enc, err := json.Marshal(c.handshakeRequest)
	if err != nil {
		return err
	}

	p, err := c.packetEncoder.Encode(packet.Handshake, enc)
	if err != nil {
		return err
	}

	_, err = c.conn.Write(p)
	return err
}

func (c *Client) handleHandshakeResponse() error {
	buf := bytes.NewBuffer(nil)
	packets, err := c.readPackets(buf)
	if err != nil || len(packets) == 0 {
		return err
	}

	handshakePacket := packets[0]
	if handshakePacket.Type != packet.Handshake {
		return fmt.Errorf("got first packet from server that is not a handshake, aborting")
	}

	handshake := &HandshakeResponse{}
	if compression.IsCompressed(handshakePacket.Data) {
		handshakePacket.Data, err = compression.InflateData(handshakePacket.Data)
		if err != nil {
			return err
		}
	}

	err = json.Unmarshal(handshakePacket.Data, handshake)
	if err != nil {
		return err
	}

	logo.Debug("got handshake from sv, data: %v", handshake)

	if handshake.Sys.Dict != nil {
		message.SetDictionary(handshake.Sys.Dict)
	}
	p, err := c.packetEncoder.Encode(packet.HandshakeAck, []byte{})
	if err != nil {
		return err
	}
	_, err = c.conn.Write(p)
	if err != nil {
		return err
	}

	atomic.StoreInt32(&c.isConnected, 1)

	go c.sendHeartbeats(handshake.Sys.Heartbeat)
	go c.handleServerMessages()
	go c.handlePackets()

	return nil
}

func (c *Client) handlePackets() {
	for {
		select {
		case p := <-c.packetChan:
			switch p.Type {
			case packet.Data:
				m, err := message.Decode(p.Data)
				if err != nil {
					logo.Info("error decoding msg from sv: %s", string(m.Data))
				}
				c.IncomingMsgChan <- m
			case packet.Kick:
				logo.Info("got kick packet from the server! disconnecting...")
				c.Disconnect()
			}
		case <-c.wc.C():
			return
		}
	}
}

func (c *Client) readPackets(buf *bytes.Buffer) ([]*packet.Packet, error) {
	// listen for sv messages
	data := make([]byte, 1024)
	n := len(data)
	var err error

	for n == len(data) {
		n, err = c.conn.Read(data)
		if err != nil {
			return nil, err
		}
		buf.Write(data[:n])
	}
	packets, err := c.packetDecoder.Decode(buf.Bytes())
	if err != nil {
		logo.Info("error decoding packet from server: %s", err.Error())
	}
	totalProcessed := 0
	for _, p := range packets {
		totalProcessed += codec.HeadLength + p.Length
	}
	buf.Next(totalProcessed)

	return packets, nil
}

func (c *Client) handleServerMessages() {
	buf := bytes.NewBuffer(nil)
	defer c.Disconnect()
	for c.IsConnected() {
		packets, err := c.readPackets(buf)
		if err != nil && c.IsConnected() {
			logo.Info(err)
			break
		}

		for _, p := range packets {
			c.packetChan <- p
		}
	}
}

func (c *Client) sendHeartbeats(interval int) {
	t := time.NewTicker(time.Duration(interval) * time.Second)
	defer func() {
		t.Stop()
		c.Disconnect()
	}()
	for {
		select {
		case <-t.C:
			p, _ := c.packetEncoder.Encode(packet.Heartbeat, []byte{})
			_, err := c.conn.Write(p)
			if err != nil {
				logo.Info("error sending heartbeat to server: %s", err.Error())
				return
			}
		case <-c.wc.C():
			return
		}
	}
}

// Disconnect disconnects the client
func (c *Client) Disconnect() {
	_ = c.wc.Close(func() error {
		if c.IsConnected() {
			atomic.StoreInt32(&c.isConnected, 0)
			_ = c.conn.Close()
		}
		return nil
	})
}

// ConnectTo connects to the server at addr, for now the only supported protocol is tcp
// if tlsConfig is sent, it connects using TLS
func (c *Client) ConnectTo(addr string, tlsConfig ...*tls.Config) error {
	var conn net.Conn
	var err error
	if len(tlsConfig) > 0 {
		conn, err = tls.Dial("tcp", addr, tlsConfig[0])
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	c.conn = conn
	c.IncomingMsgChan = make(chan *message.Message, 10)

	if err = c.handleHandshake(); err != nil {
		return err
	}

	//c.closeChan = make(chan struct{})

	return nil
}

// ConnectToWS connects using webshocket protocol
func (c *Client) ConnectToWS(addr string, path string, tlsConfig ...*tls.Config) error {
	u := url.URL{Scheme: "ws", Host: addr, Path: path}
	dialer := ws.DefaultDialer

	if len(tlsConfig) > 0 {
		dialer.TLSConfig = tlsConfig[0]
		u.Scheme = "wss"
	}

	conn, _, _, err := dialer.Dial(context.Background(), u.String())
	if err != nil {
		return err
	}

	c.conn = newClientConn(conn)
	c.IncomingMsgChan = make(chan *message.Message, 10)

	if err = c.handleHandshake(); err != nil {
		return err
	}

	return nil
}

func (c *Client) handleHandshake() error {
	if err := c.sendHandshakeRequest(); err != nil {
		return err
	}

	if err := c.handleHandshakeResponse(); err != nil {
		return err
	}
	return nil
}

// SendRequest sends a request to the server
func (c *Client) SendRequest(route string, data []byte) (uint, error) {
	return c.sendMsg(message.Request, route, data)
}

// SendNotify sends a notify to the server
func (c *Client) SendNotify(route string, data []byte) error {
	_, err := c.sendMsg(message.Notify, route, data)
	return err
}

func (c *Client) buildPacket(msg message.Message) ([]byte, error) {
	encMsg, err := c.messageEncoder.Encode(&msg)
	if err != nil {
		return nil, err
	}
	p, err := c.packetEncoder.Encode(packet.Data, encMsg)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// sendMsg sends the request to the server
func (c *Client) sendMsg(msgType message.Type, route string, data []byte) (uint, error) {
	// TODO mount msg and encode
	m := message.Message{
		Type:  msgType,
		Id:    uint(atomic.AddUint32(&c.nextId, 1)),
		Route: route,
		Data:  data,
		Err:   false,
	}
	p, err := c.buildPacket(m)
	if msgType == message.Request {

	}

	if err != nil {
		return m.Id, err
	}
	_, err = c.conn.Write(p)
	return m.Id, err
}
