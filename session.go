package bugfly

import (
	"context"
	"github.com/lixianmin/gonsole/logger"
	"github.com/lixianmin/gonsole/bugfly/conn/codec"
	"github.com/lixianmin/gonsole/bugfly/conn/message"
	"github.com/lixianmin/gonsole/bugfly/route"
	"github.com/lixianmin/gonsole/bugfly/serialize"
	"github.com/lixianmin/got/loom"
	"sync"
	"sync/atomic"
	"time"
)

/********************************************************************
created:    2020-08-28
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

var (
	idGenerator int64 = 0
)

type (
	commonSessionArgs struct {
		packetEncoder  codec.PacketEncoder
		packetDecoder  codec.PacketDecoder
		messageEncoder message.Encoder
		serializer     serialize.Serializer

		heartbeatTimeout      time.Duration
		heartbeatPacketData   []byte
		handshakeResponseData []byte
	}

	Session struct {
		commonSessionArgs
		id           int64
		conn         PlayerConn
		sendingChan  chan sendingItem
		receivedChan chan receivedItem
		lastAt       int64 // last heartbeat unix time stamp
		wc           loom.WaitClose

		onClosedCallbacks []func()
		lock              sync.Mutex
	}

	receivedItem struct {
		ctx   context.Context
		route *route.Route
		msg   *message.Message
	}

	sendingItem struct {
		ctx  context.Context
		data []byte
		err  error
	}

	// 未编码消息
	sendingInfo struct {
		ctx     context.Context
		typ     message.Type // message type
		route   string       // message route (push)
		mid     uint         // response message id (response)
		payload interface{}  // payload
		err     bool         // if its an error message
	}
)

func NewSession(conn PlayerConn, args commonSessionArgs) *Session {
	const bufferSize = 16
	var agent = &Session{
		commonSessionArgs: args,
		id:                atomic.AddInt64(&idGenerator, 1),
		conn:              conn,
		sendingChan:       make(chan sendingItem, bufferSize),
		receivedChan:      make(chan receivedItem, bufferSize),
		lastAt:            time.Now().Unix(),
	}

	loom.Go(agent.goReceive)
	loom.Go(agent.goSend)
	loom.Go(agent.goProcess)
	return agent
}

func (my *Session) Close() {
	my.wc.Close(func() {
		_ = my.conn.Close()
		my.invokeOnClosedCallbacks()
	})
}

func (my *Session) invokeOnClosedCallbacks() {
	var cloned = make([]func(), len(my.onClosedCallbacks))

	// 单独clone一份出来，因为callback的方法体调用了哪些内容未知，防止循环调用导致死循环
	my.lock.Lock()
	for i, callback := range my.onClosedCallbacks {
		cloned[i] = callback
	}
	my.lock.Unlock()

	defer func() {
		if r := recover(); r != nil {
			logger.Info("[invokeOnClosedCallbacks()] panic: r=%v", r)
		}
	}()

	for _, callback := range cloned {
		callback()
	}
}

func (my *Session) OnClosed(callback func()) {
	if callback == nil {
		return
	}

	my.lock.Lock()
	my.onClosedCallbacks = append(my.onClosedCallbacks, callback)
	my.lock.Unlock()
}

func (my *Session) GetSessionId() int64 {
	return my.id
}
