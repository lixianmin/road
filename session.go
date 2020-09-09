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

todo 关于是否添加SendCallback() 的考虑：
	正面：1. 异步转同步；2. 分帧削峰
	负面：1. 这个方法只对业务有可能有用，但对网络库本身并没有意义；2. 必须谨慎使用，过长的处理时间会影响后续网络消息处理，可能导致链接超时（当然你可以选择不用）

todo 1. 将loom.Map分uint32, uint64与string，分类型初始化，这对速度和内存占用将会是一个bonus，本来我们也只支持这几种类型；
	 2. 将初始化的大小减小，但增加一个只允许调用一次的分片方法，这个跟直接加一个NewMap(shardingCount)初始化方法相比孰优孰劣？减小初始化大小会降低内存使用量（为什么印象里实测并没有）
		但是，会增加使用复杂度，因为大多数人都只使用默认的设置；而强迫使用NewMap(shardingCount)方法会强制所有的人都使用这个构造函数，这其实有些不方便使用。

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

// Close()方法可以被多次调用，只触发一次OnClosed事件
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

// 需要保证OnClosed事件在任何情况下都会有且仅有一次触发：无论是主动断开，还是意外断开链接；无论client端有没有因为网络问题收到回复消息
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
