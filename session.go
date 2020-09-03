package road

import (
	"context"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/acceptor"
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/logger"
	"github.com/lixianmin/road/route"
	"github.com/lixianmin/road/serialize"
	"net"
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
		conn         acceptor.PlayerConn
		attachment   *Attachment
		sendingChan  chan []byte
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
)

func NewSession(conn acceptor.PlayerConn, args commonSessionArgs) *Session {
	const bufferSize = 16
	var agent = &Session{
		commonSessionArgs: args,
		id:                atomic.AddInt64(&idGenerator, 1),
		conn:              conn,
		attachment:        &Attachment{},
		sendingChan:       make(chan []byte, bufferSize),
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
		my.attachment.dispose()
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

func (my *Session) Id() int64 {
	return my.id
}

func (my *Session) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}

func (my *Session) Attachment() *Attachment {
	return my.attachment
}
