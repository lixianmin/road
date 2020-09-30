package road

import (
	"context"
	"errors"
	"fmt"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/got/timex"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/epoll"
	"github.com/lixianmin/road/ifs"
	"github.com/lixianmin/road/logger"
	"github.com/lixianmin/road/route"
	"github.com/lixianmin/road/serialize"
	"github.com/lixianmin/road/util"
	"reflect"
	"time"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func (my *Session) goLoop(later loom.Later) {
	defer my.Close()

	var receivedChan = my.conn.GetReceivedChan()
	var closeChan = my.wc.C()
	var app = my.app
	var heartbeatTimer = app.wheelSecond.NewTimer(app.heartbeatTimeout)

	var fetus = &sessionFetus{
		lastAt:        timex.NowUnix(),
		deltaDeadline: int64(3 * app.heartbeatTimeout / time.Second),
	}

	for {
		select {
		case <-heartbeatTimer.C:
			heartbeatTimer.Reset()

			if err := my.onHeartbeat(fetus); err != nil {
				logger.Info(err.Error())
				return
			}
		case data := <-my.sendingChan:
			if _, err := my.conn.Write(data); err != nil {
				logger.Info("Failed to write in conn: %s", err.Error())
				return
			}
		case msg := <-receivedChan:
			fetus.lastAt = timex.NowUnix()
			if err := my.onReceivedMessage(fetus, msg); err != nil {
				logger.Info(err.Error())
				return
			}
		case task := <-my.tasks.C:
			_ = task.Do(my)
		case <-closeChan:
			return
		}
	}
}

func (my *Session) onHeartbeat(fetus *sessionFetus) error {
	// 如果在一个心跳时间后还没有收到握手消息，就断开链接。
	// 登录验证之类的事情是在机会在onHandShaken事件中验证的
	if !fetus.isHandshakeReceived {
		return errors.New("don't received handshake, disconnect")
	}

	deadline := timex.NowUnix() - fetus.deltaDeadline
	if fetus.lastAt < deadline {
		return fmt.Errorf("session heartbeat timeout, lastAt=%d, deadline=%d", fetus.lastAt, deadline)
	}

	// 发送心跳包，如果网络是通的，收到心跳返回时会刷新 lastAt
	if _, err := my.conn.Write(my.app.heartbeatPacketData); err != nil {
		return fmt.Errorf("failed to write in conn: %s", err.Error())
	}

	return nil
}

func (my *Session) onReceivedMessage(fetus *sessionFetus, msg epoll.Message) error {
	var err = msg.Err
	if err != nil {
		var err1 = fmt.Errorf("error reading next available message: %s", err.Error())
		return err1
	}

	packets, err := my.app.packetDecoder.Decode(msg.Data)
	if err != nil {
		var err1 = fmt.Errorf("failed to decode message: %s", err.Error())
		return err1
	}

	// process all packet
	for i := range packets {
		var p = packets[i]
		switch p.Type {
		case packet.Handshake:
			my.onReceivedHandshake(fetus, p)
		case packet.HandshakeAck:
			// handshake的流程是 client (request) --> server (response) --> client (ack) --> server (received ack)
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

// 如果长时间收不到握手消息，服务器会主动断开链接
func (my *Session) onReceivedHandshake(args *sessionFetus, p *packet.Packet) {
	args.isHandshakeReceived = true
	my.sendBytes(my.app.handshakeResponseData)
	my.onHandShaken.Invoke()
}

func (my *Session) onReceivedData(p *packet.Packet) error {
	item, err := my.decodeReceivedData(p);
	if err != nil {
		var err1 = fmt.Errorf("failed to process packet: %s", err.Error())
		return err1
	}

	handler, err := my.app.getHandler(item.route)
	if err != nil {
		return err
	}

	payload, err := processReceivedData(item, handler, my.app.serializer, my.app.hookCallback)
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

func processReceivedData(data receivedItem, handler *component.Handler, serializer serialize.Serializer, hookCallback HookFunc) ([]byte, error) {
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

	resp, err := hookCallback(func() (i interface{}, e error) {
		return util.Pcall(handler.Method, args)
	})

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
