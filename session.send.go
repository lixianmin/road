package bugfly

import (
	"context"
	"fmt"
	"github.com/lixianmin/gonsole/logger"
	"github.com/lixianmin/gonsole/bugfly/conn/message"
	"github.com/lixianmin/gonsole/bugfly/conn/packet"
	"github.com/lixianmin/gonsole/bugfly/util"
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
	return my.send(sendingInfo{typ: message.Push, route: route, payload: v})
}

func (my *Session) responseMID(ctx context.Context, mid uint, payload interface{}) error {
	return my.send(sendingInfo{ctx: ctx, typ: message.Response, mid: mid, payload: payload, err: false})
}

func (my *Session) send(info sendingInfo) error {
	defer func() {
		if e := recover(); e != nil {
			logger.Info(e)
		}
	}()

	payload, err := util.SerializeOrRaw(my.serializer, info.payload)
	if err != nil {
		return err
	}

	// construct message and encode
	m := &message.Message{
		Type:  info.typ,
		Data:  payload,
		Route: info.route,
		ID:    info.mid,
		Err:   info.err,
	}

	// packet encode
	p, err := my.packetEncodeMessage(m)
	if err != nil {
		return err
	}

	item := sendingItem{
		ctx:  info.ctx,
		data: p,
	}

	if info.err {
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
