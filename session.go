package road

import (
	"context"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/acceptor"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/route"
	"net"
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
	Session struct {
		commonSessionArgs
		id           int64
		conn         acceptor.PlayerConn
		attachment   *Attachment
		sendingChan  chan []byte
		receivedChan chan receivedItem
		lastAt       int64 // last heartbeat unix time stamp
		wc           loom.WaitClose

		onHandShaken delegate
		onClosed     delegate
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
		my.onClosed.Invoke()
	})
}

func (my *Session) OnHandShaken(handler func()) {
	my.onHandShaken.Add(handler)
}

func (my *Session) OnClosed(handler func()) {
	my.onClosed.Add(handler)
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
