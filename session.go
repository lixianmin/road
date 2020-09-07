package road

import (
	"context"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/epoll"
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
		id          int64
		conn        epoll.PlayerConn
		attachment  *Attachment
		sendingChan chan []byte
		lastAt      int64 // last heartbeat unix time stamp
		wc          loom.WaitClose

		onHandShaken delegate
		onClosed     delegate
	}

	receivedItem struct {
		ctx   context.Context
		route *route.Route
		msg   *message.Message
	}
)

func NewSession(conn epoll.PlayerConn, args commonSessionArgs) *Session {
	const bufferSize = 16
	var agent = &Session{
		commonSessionArgs: args,
		id:                atomic.AddInt64(&idGenerator, 1),
		conn:              conn,
		attachment:        &Attachment{},
		sendingChan:       make(chan []byte, bufferSize),
		lastAt:            time.Now().Unix(),
	}

	loom.Go(agent.goLoop)
	return agent
}

func (my *Session) Close() error {
	return my.wc.Close(func() error {
		var err = my.conn.Close()
		my.attachment.dispose()
		my.onClosed.Invoke()
		return err
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
