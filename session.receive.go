package road

import (
	"context"
	"fmt"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/epoll"
	"github.com/lixianmin/road/ifs"
	"github.com/lixianmin/road/logger"
	"github.com/lixianmin/road/route"
	"github.com/lixianmin/road/serialize"
	"github.com/lixianmin/road/service"
	"github.com/lixianmin/road/util"
	"reflect"
	"time"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func (my *Session) goLoop(later *loom.Later) {
	defer my.Close()

	var receivedChan = my.conn.GetReceivedChan()
	var heartbeatTicker = later.NewTicker(my.heartbeatTimeout)
	var closeChan = my.wc.C()
	var lastAt = time.Now().Unix() // last heartbeat unix time stamp

	for {
		select {
		case <-heartbeatTicker.C:
			deadline := time.Now().Add(-2 * my.heartbeatTimeout).Unix()
			if lastAt < deadline {
				logger.Info("Session heartbeat timeout, LastTime=%d, Deadline=%d", lastAt, deadline)
				return
			}

			if _, err := my.conn.Write(my.heartbeatPacketData); err != nil {
				logger.Info("Failed to write in conn: %s", err.Error())
				return
			}
		case data := <-my.sendingChan:
			if _, err := my.conn.Write(data); err != nil {
				logger.Info("Failed to write in conn: %s", err.Error())
				return
			}
		case msg := <-receivedChan:
			lastAt = time.Now().Unix()
			if err := my.onReceivedMessage(msg); err != nil {
				logger.Info(err.Error())
				return
			}
		case <-closeChan:
			return
		}
	}
}

func (my *Session) onReceivedMessage(msg epoll.Message) error {
	var err = msg.Err
	if err != nil {
		var err1 = fmt.Errorf("error reading next available message: %s", err.Error())
		return err1
	}

	packets, err := my.packetDecoder.Decode(msg.Data)
	if err != nil {
		var err1 = fmt.Errorf("failed to decode message: %s", err.Error())
		return err1
	}

	// process all packet
	for i := range packets {
		var p = packets[i]
		switch p.Type {
		case packet.Handshake:
			my.onReceivedHandshake(p)
		case packet.HandshakeAck:
			logger.Debug("Receive handshake ACK")
		case packet.Data:
			if err := my.onReceivedData(p); err != nil {
				return err
			}
		case packet.Heartbeat:
			// expected
		}
	}

	return nil
}

func (my *Session) onReceivedHandshake(p *packet.Packet) {
	my.sendBytes(my.handshakeResponseData)
	my.onHandShaken.Invoke()
}

func (my *Session) onReceivedData(p *packet.Packet) error {
	item, err := my.decodeReceivedData(p);
	if err != nil {
		var err1 = fmt.Errorf("failed to process packet: %s", err.Error())
		return err1
	}

	payload, err := processReceivedData(item, my.serializer)
	if item.msg.Type != message.Notify {
		var msg = message.Message{Type: message.Response, ID: item.msg.ID, Data: payload}
		_ = my.sendMessageMayError(msg, err)
	}

	return nil
}

func (my *Session) decodeReceivedData(p *packet.Packet) (receivedItem, error) {
	msg, err := message.Decode(p.Data)
	if err != nil {
		return receivedItem{}, err
	}

	r, err := route.Decode(msg.Route)
	if err != nil {
		return receivedItem{}, err
	}

	var ctx = context.WithValue(context.Background(), ifs.CtxKeySession, my)

	var item = receivedItem{
		ctx:   ctx,
		route: r,
		msg:   msg,
	}

	return item, nil
}

func processReceivedData(data receivedItem, serializer serialize.Serializer) ([]byte, error) {
	handler, err := service.GetHandler(data.route)
	if err != nil {
		return nil, err
	}

	// First unmarshal the handler arg that will be passed to
	// both handler and pipeline functions
	arg, err := unmarshalHandlerArg(handler, serializer, data.msg.Data)
	if err != nil {
		return nil, err
	}

	var args []reflect.Value
	if arg != nil {
		args = []reflect.Value{handler.Receiver, reflect.ValueOf(data.ctx), reflect.ValueOf(arg)}
	} else {
		args = []reflect.Value{handler.Receiver, reflect.ValueOf(data.ctx)}
	}

	resp, err := util.Pcall(handler.Method, args)
	if err != nil {
		return nil, err
	}

	ret, err := util.SerializeOrRaw(serializer, resp)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func unmarshalHandlerArg(handler *component.Handler, serializer serialize.Serializer, payload []byte) (interface{}, error) {
	if handler.IsRawArg {
		return payload, nil
	}

	var arg interface{}
	if handler.Type != nil {
		arg = reflect.New(handler.Type.Elem()).Interface()
		err := serializer.Unmarshal(payload, arg)
		if err != nil {
			return nil, err
		}
	}

	return arg, nil
}
