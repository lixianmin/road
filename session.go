package road

import (
	"context"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/epoll"
	"github.com/lixianmin/road/logger"
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
	globalIdGenerator int64 = 0
)

type (
	Session struct {
		app         *App
		id          int64
		conn        epoll.PlayerConn
		attachment  *Attachment
		sendingChan chan []byte
		wc          loom.WaitClose
		tasks       *loom.TaskQueue

		onHandShaken delegate
		onClosed     delegate
	}

	sessionFetus struct {
		isHandshakeReceived bool          // 是否接收到handshake消息
		lastAt              time.Time     // 最后一时收到数据的时间戳
		heartbeatTimeout    time.Duration // 用于判断心跳是否超时
		rateLimitTokens     int32         // 限流令牌
		rateLimitWindow     int32         // 限流窗口
	}

	receivedItem struct {
		ctx   context.Context
		route *route.Route
		msg   *message.Message
	}
)

func NewSession(app *App, conn epoll.PlayerConn) *Session {
	var my = &Session{
		app:         app,
		id:          atomic.AddInt64(&globalIdGenerator, 1),
		conn:        conn,
		attachment:  &Attachment{},
		sendingChan: make(chan []byte, app.sendingChanSize),
	}

	my.tasks = loom.NewTaskQueue(loom.WithSize(app.taskQueueSize), loom.WithCloseChan(my.wc.C()))
	logger.Info("create session(%d)", my.id)
	loom.Go(my.goSessionLoop)
	return my
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
func (my *Session) OnHandShaken(handler func()) {
	my.onHandShaken.Add(handler)
}

// 需要保证OnClosed事件在任何情况下都会有且仅有一次触发：无论是主动断开，还是意外断开链接；无论client端有没有因为网络问题收到回复消息
func (my *Session) OnClosed(handler func()) {
	my.onClosed.Add(handler)
}

// 在session中加入SendCallback()的相关权衡？
//
// 正面：
// 1. 异步转同步
// 2. 分帧削峰
// 3. player类不再需要独立的goroutine，至少节约2~8KB的goroutine初始内存，这比TaskQueue占用的内存要多得多
//
// 负面：
// 1. 这个方法只对业务有可能有用，但对网络库本身并没有意义；
// 2. 必须谨慎使用，过长的处理时间会影响后续网络消息处理，可能导致链接超时（当然你可以选择不用）
func (my *Session) SendCallback(handler loom.TaskHandler) loom.ITask {
	return my.tasks.SendCallback(handler)
}

// 延迟任务
func (my *Session) SendDelayed(delayed time.Duration, handler loom.TaskHandler) {
	my.tasks.SendDelayed(delayed, handler)
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
