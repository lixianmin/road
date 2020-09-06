package road

import (
	"context"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/ifs"
	"github.com/lixianmin/road/logger"
	"github.com/lixianmin/road/route"
	"sync/atomic"
	"time"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func (my *Session) goReceive(later *loom.Later) {
	defer my.Close()

	var receivedChan = my.conn.GetReceivedChan()
	var heartbeatTicker = later.NewTicker(my.heartbeatTimeout)

	for {
		select {
		case <-heartbeatTicker.C:
			deadline := time.Now().Add(-2 * my.heartbeatTimeout).Unix()
			if atomic.LoadInt64(&my.lastAt) < deadline {
				logger.Info("Session heartbeat timeout, LastTime=%d, Deadline=%d", atomic.LoadInt64(&my.lastAt), deadline)
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
			var err = msg.Err
			if err != nil {
				logger.Info("Error reading next available message: %s", err.Error())
				return
			}

			packets, err := my.packetDecoder.Decode(msg.Data)
			if err != nil {
				logger.Info("Failed to decode message: %s", err.Error())
				return
			}

			if len(packets) < 1 {
				logger.Warn("Read no packets, data: %v", msg)
				continue
			}

			// process all packet
			for i := range packets {
				var p = packets[i]
				var item, err = my.onReceivedPacket(p)
				if err != nil {
					logger.Info("Failed to process packet: %s", err.Error())
					return
				}

				if p.Type == packet.Data {
					my.processReceived(item)
				}

				atomic.StoreInt64(&my.lastAt, time.Now().Unix())
			}
		}
	}
}

func (my *Session) onReceivedPacket(p *packet.Packet) (receivedItem, error) {
	switch p.Type {
	case packet.Handshake:
		my.onReceivedHandshake(p)
	case packet.HandshakeAck:
		logger.Debug("Receive handshake ACK")
	case packet.Data:
		return my.onReceivedDataPacket(p)
	case packet.Heartbeat:
		// expected
	}

	return receivedItem{}, nil
}

func (my *Session) onReceivedHandshake(p *packet.Packet) {
	my.sendBytes(my.handshakeResponseData)
	my.onHandShaken.Invoke()
}

func (my *Session) onReceivedDataPacket(p *packet.Packet) (receivedItem, error) {
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
