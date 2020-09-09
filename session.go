package road

import (
	"context"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/epoll"
	"github.com/lixianmin/road/route"
	"net"
	"sync/atomic"
)

/********************************************************************
created:    2020-08-28
author:     lixianmin

todo 关于是否添加SendCallback() 的考虑：
	正面：1. 异步转同步；2. 分帧削峰
	负面：1. 这个方法只对业务有可能有用，但对网络库本身并没有意义；2. 必须谨慎使用，过长的处理时间会影响后续网络消息处理，可能导致链接超时（当然你可以选择不用）

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
		wc          loom.WaitClose

		onHandShaken delegate
		onClosed     delegate
	}

	sessionLoopArgs struct {

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
	}

	loom.Go(agent.goLoop)
	return agent
}

// Close()方法可以被多次调用，只触发一次OnClosed事件
func (my *Session) Close() error {
	return my.wc.Close(func() error {
		var err = my.conn.Close()
		my.attachment.dispose()
		my.onClosed.Invoke()
		return err
	})
}

// 握手事件：收到握手消息后触发。
// todo 如果长时间收不到握手消息，应该主动断开链接
func (my *Session) OnHandShaken(handler func()) {
	my.onHandShaken.Add(handler)
}

// 需要保证OnClosed事件在任何情况下都会有且仅有一次触发：无论是主动断开，还是意外断开链接；无论client端有没有因为网络问题收到回复消息
func (my *Session) OnClosed(handler func()) {
	my.onClosed.Add(handler)
}

// 全局唯一id
func (my *Session) Id() int64 {
	return my.id
}

func (my *Session) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}

func (my *Session) Attachment() *Attachment {
	return my.attachment
}
