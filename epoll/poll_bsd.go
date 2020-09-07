// +build darwin netbsd freebsd openbsd dragonfly

package epoll

import (
	"github.com/gobwas/ws/wsutil"
	"github.com/lixianmin/got/loom"
	"io"
	"net"
	"sync"
	"syscall"
)

/********************************************************************
created:    2020-09-06
author:     lixianmin

参考:
1. https://github.com/smallnest/epoller/blob/master/epoll_bsd.go
2. https://github.com/eranyanay/1m-go-websockets/blob/master/4_optimize_gobwas/server.go

Copyright (C) - All Rights Reserved
*********************************************************************/

type Poll struct {
	receivedChanLen int
	fd              int
	connections     loom.Map
	wc              loom.WaitClose

	changes struct {
		sync.Mutex // 没有必要使用RWMutex，因为只有一个goLoop()在读
		d          []syscall.Kevent_t
	}
}

type loopArgs struct {
	snapshot []syscall.Kevent_t
	events   []syscall.Kevent_t
	timeout  syscall.Timespec
}

func newPoll(pollBufferSize int, receivedChanLen int) *Poll {
	fd, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}

	if _, err := syscall.Kevent(fd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil); err != nil {
		panic(err)
	}

	var poll = &Poll{
		receivedChanLen: receivedChanLen,
		fd:              fd,
	}

	poll.changes.d = make([]syscall.Kevent_t, 0, pollBufferSize)
	loom.Go(func(later *loom.Later) {
		poll.goLoop(later, pollBufferSize)
	})

	return poll
}

func (my *Poll) goLoop(later *loom.Later, bufferSize int) {
	defer my.Close()
	var args = &loopArgs{
		snapshot: make([]syscall.Kevent_t, bufferSize),
		events:   make([]syscall.Kevent_t, bufferSize),
		timeout:  syscall.NsecToTimespec(1e7), // 将超时时间改为10ms，这其实是上一轮没有数据时，下一轮fd们的最长等待时间
	}

	for {
		select {
		case <-my.wc.C():
			return
		default:
			my.pollData(args)
		}
	}
}

func (my *Poll) Close() error {
	return my.wc.Close(func() error {
		my.changes.Lock()
		my.changes.d = nil
		my.connections = loom.Map{}
		my.changes.Unlock()

		var err = syscall.Close(my.fd)
		return err
	})
}

// 记录当前活跃的链接，出错后通过Remove方法移除
func (my *Poll) add(conn net.Conn) *WSConn {
	var fd = socketFD(conn)

	var event = syscall.Kevent_t{Ident: uint64(fd), Flags: syscall.EV_ADD | syscall.EV_EOF, Filter: syscall.EVFILT_READ}
	var receivedChan = make(chan Message, my.receivedChanLen)
	var playerConn *WSConn

	my.changes.Lock()
	{
		my.changes.d = append(my.changes.d, event)
		playerConn = &WSConn{
			fd:           fd,
			conn:         conn,
			receivedChan: receivedChan,
		}
		my.connections.Put(fd, playerConn)
	}
	my.changes.Unlock()

	return playerConn
}

func (my *Poll) remove(item *WSConn) error {
	my.changes.Lock()
	{
		// 找到fd出现的位置
		var changes = my.changes.d
		var count = len(my.changes.d)
		var idxFind = -1
		for i := 0; i < count; i++ {
			if changes[i].Ident == uint64(item.fd) {
				idxFind = i
				break
			}
		}

		if idxFind >= 0 {
			for i, j := idxFind+1, idxFind; i < count; i, j = i+1, j+1 {
				changes[j] = changes[i]
			}
			my.changes.d = changes[:count-1]
			my.connections.Remove(item.fd)
		}
	}
	my.changes.Unlock()

	// 关闭链接；关闭chan
	var err = item.Close()
	return err
}

func (my *Poll) takeSnapshot(args *loopArgs) {
	my.changes.Lock()
	var snapCount = len(my.changes.d)
	args.snapshot = args.snapshot[:snapCount]
	for i := 0; i < snapCount; i++ {
		args.snapshot[i] = my.changes.d[i]
	}
	my.changes.Unlock()
}

func (my *Poll) pollData(args *loopArgs) {
retry:
	my.takeSnapshot(args)
	num, err := syscall.Kevent(my.fd, args.snapshot, args.events, &args.timeout)

	if err != nil {
		if err == syscall.EINTR {
			goto retry
		}
		return
	}

	for i := 0; i < num; i++ {
		var ident = args.events[i].Ident
		var item = my.connections.Get1(ident).(*WSConn)
		var conn = item.conn

		// EOF
		if (args.events[i].Flags & syscall.EV_EOF) == syscall.EV_EOF {
			item.receivedChan <- Message{Err: io.EOF}
			_ = my.remove(item)
			continue
		}

		var data, _, err = wsutil.ReadClientData(conn)
		if err != nil {
			item.receivedChan <- Message{Err: err}
			_ = my.remove(item)
			continue
		}

		if err := checkReceivedMsgBytes(data); err != nil {
			item.receivedChan <- Message{Err: err}
			_ = my.remove(item)
			continue
		}

		item.receivedChan <- Message{Data: data}
	}
}
