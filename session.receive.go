package road

import (
	"context"
	"errors"
	"fmt"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/got/mathx"
	"github.com/lixianmin/got/timex"
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/epoll"
	"github.com/lixianmin/road/ifs"
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

func (my *sessionImpl) goSessionLoop(later loom.Later) {
	defer my.Close()

	var receivedChan = my.conn.GetReceivedChan()
	var closeChan = my.wc.C()
	var app = my.app

	var heartbeatInterval = app.heartbeatInterval
	var heartbeatTimer = app.wheelSecond.NewTimer(heartbeatInterval)
	var stepRateLimitTokens = mathx.MaxInt32(1, int32(float64(heartbeatInterval)/float64(time.Second)*float64(app.rateLimitBySecond)))

	var fetus = &sessionFetus{
		lastAt:           time.Now(),
		heartbeatTimeout: heartbeatInterval * 3,
		rateLimitTokens:  stepRateLimitTokens,
		rateLimitWindow:  2 * stepRateLimitTokens,
	}

	for {
		select {
		case <-heartbeatTimer.C:
			heartbeatTimer.Reset()

			// 使用时间窗口限制令牌数
			fetus.rateLimitTokens = mathx.MinInt32(fetus.rateLimitWindow, fetus.rateLimitTokens+stepRateLimitTokens)

			if err := my.onHeartbeat(fetus); err != nil {
				logo.Info("close session(%d) by onHeartbeat(), err=%q", my.id, err)
				return
			}
		case msg := <-receivedChan:
			fetus.lastAt = time.Now()
			fetus.rateLimitTokens--
			if err := my.onReceivedMessage(fetus, msg); err != nil {
				logo.Info("close session(%d) by onReceivedMessage(), err=%q", my.id, err)
				return
			}
		//case task := <-my.tasks.C:
		//	_ = task.Do(my)
		case <-closeChan:
			logo.Info("close session(%d) by calling session.Close()", my.id)
			return
		}
	}
}

func (my *sessionImpl) onHeartbeat(fetus *sessionFetus) error {
	// 如果在一个心跳时间后还没有收到握手消息，就断开链接。
	// 登录验证之类的事情是在机会在onHandShaken事件中验证的
	if !fetus.isHandshakeReceived {
		return errors.New("don't received handshake, disconnect")
	}

	var passedTime = time.Now().Sub(fetus.lastAt)
	if passedTime > fetus.heartbeatTimeout {
		return fmt.Errorf("session heartbeat timeout, lastAt=%q, heartbeatTimeout=%s", fetus.lastAt.Format(timex.Layout), fetus.heartbeatTimeout)
	}

	// 发送心跳包，如果网络是通的，收到心跳返回时会刷新 lastAt
	if err := my.writeBytes(my.app.heartbeatPacketData); err != nil {
		return fmt.Errorf("failed to write in conn: %s", err.Error())
	}

	// 注意：libpitaya的heartbeat部分好像是问题的，只能在应用层自己做ping/pong
	//logo.Debug("session(%d) sent heartbeat", my.id)
	return nil
}

func (my *sessionImpl) onReceivedMessage(fetus *sessionFetus, msg epoll.Message) error {
	var err = msg.Err
	if err != nil {
		return msg.Err
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
			if err := my.onReceivedHandshake(fetus, p); err != nil {
				return err
			}
		case packet.HandshakeAck:
			// handshake的流程是 client (request) --> server (response) --> client (ack) --> server (received ack)
			logo.Debug("session(%d) received handshake ACK", my.id)
		case packet.Data:
			if err := my.onReceivedData(fetus, p); err != nil {
				return err
			}
		case packet.Heartbeat:
			//logo.Debug("session(%d) received heartbeat", my.id)
		}
	}

	return nil
}

// 如果长时间收不到握手消息，服务器会主动断开链接
func (my *sessionImpl) onReceivedHandshake(fetus *sessionFetus, p *packet.Packet) error {
	fetus.isHandshakeReceived = true
	var err = my.writeBytes(my.app.handshakeResponseData)
	if err == nil {
		my.onHandShaken.Invoke()
	}

	return err
}

func (my *sessionImpl) onReceivedData(fetus *sessionFetus, p *packet.Packet) error {
	item, err := my.decodeReceivedData(p)
	if err != nil {
		var err1 = fmt.Errorf("failed to process packet: %s", err.Error())
		return err1
	}

	// 如果令牌数耗尽，则拒绝处理，并给客户端报错
	needReply := item.msg.Type != message.Notify
	//logo.Debug("needReplay=%v, message=%v", needReply, item.msg)

	if fetus.rateLimitTokens <= 0 {
		if needReply {
			var msg = message.Message{Type: message.Response, Id: item.msg.Id}
			var data, err1 = my.encodeMessageMayError(msg, ErrTriggerRateLimit)
			if err1 != nil {
				return err1
			}

			var err2 = my.writeBytes(data)
			if err2 != nil {
				return err2
			}
		}

		// 如果单位时间内消耗令牌太多，则直接断开网络
		if fetus.rateLimitTokens <= -fetus.rateLimitWindow {
			return ErrKickedByRateLimit
		}

		return nil
	}

	// 取handler，准备处理协议
	handler, err := my.app.getHandler(item.route)
	if err != nil {
		return err
	}

	payload, err := processReceivedData(item, handler, my.app.serializer, my.app.hookCallback)
	if needReply {
		var msg = message.Message{Type: message.Response, Id: item.msg.Id, Data: payload}
		var data, err1 = my.encodeMessageMayError(msg, err)
		if err1 != nil {
			return err1
		}

		return my.writeBytes(data)
	}

	return nil
}

func (my *sessionImpl) decodeReceivedData(p *packet.Packet) (receivedItem, error) {
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
		return util.PCall(handler.Method, args)
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
