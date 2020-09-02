package bugfly

import (
	"fmt"
	"github.com/lixianmin/bugfly/conn/message"
	"github.com/lixianmin/bugfly/conn/packet"
	"github.com/lixianmin/bugfly/logger"
	"github.com/lixianmin/bugfly/util"
	"github.com/lixianmin/got/loom"
	"sync/atomic"
	"time"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func (my *Session) goSend(later *loom.Later) {
	defer my.Close()
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
		case item := <-my.sendingChan:
			if _, err := my.conn.Write(item.data); err != nil {
				logger.Info("Failed to write in conn: %s", err.Error())
				return
			}
		case <-my.wc.C():
			return
		}
	}
}

func (my *Session) Push(route string, v interface{}) error {
	var payload, err = util.SerializeOrRaw(my.serializer, v)
	var info = sendingInfo{typ: message.Push, route: route, payload: payload}
	return my.sendMayError(info, err)
}

func (my *Session) sendMayError(info sendingInfo, err error) error {
	if err != nil {
		info.hasErr = true
		logger.Info("process failed, route=%s, err=%q", info.route, err.Error())

		var err1 error
		info.payload, err1 = util.SerializeOrRaw(my.serializer, err)
		if err1 != nil {
			logger.Info("serialize failed, route=%s, err1=%q", info.route, err1.Error())
			return err1
		}
	}

	err2 := my.send(info)
	if err2 != nil {
		logger.Info("send failed, route=%s, err2=%q", info.route, err2.Error())
		return err2
	}

	return nil
}

func (my *Session) send(info sendingInfo) error {
	defer func() {
		if e := recover(); e != nil {
			logger.Info(e)
		}
	}()

	// construct message and encode
	msg := &message.Message{
		Type:  info.typ,
		Data:  info.payload,
		Route: info.route,
		ID:    info.mid,
		Err:   info.hasErr,
	}

	// packet encode
	p, err := my.packetEncodeMessage(msg)
	if err != nil {
		return err
	}

	item := sendingItem{
		ctx:  info.ctx,
		data: p,
	}

	if info.hasErr {
		item.err = fmt.Errorf("has pending error")
	}

	select {
	case <-my.wc.C():
	case my.sendingChan <- item:
	}

	return nil
}

func (my *Session) packetEncodeMessage(msg *message.Message) ([]byte, error) {
	data, err := my.messageEncoder.Encode(msg)
	if err != nil {
		return nil, err
	}

	// packet encode
	p, err := my.packetEncoder.Encode(packet.Data, data)
	if err != nil {
		return nil, err
	}

	return p, nil
}
