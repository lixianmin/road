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
	fd          int
	connections loom.Map
	wc          loom.WaitClose

	changes struct {
		sync.RWMutex
		d []syscall.Kevent_t
	}
}

func NewPoll(bufferSize int) *Poll {
	if bufferSize <= 0 {
		bufferSize = 128
	}

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
		fd: fd,
	}

	poll.changes.d = make([]syscall.Kevent_t, 0, bufferSize)
	loom.Go(func(later *loom.Later) {
		poll.goLoop(later, bufferSize)
	})

	return poll
}

func (my *Poll) goLoop(later *loom.Later, bufferSize int) {
	defer my.Close()
	var events = make([]syscall.Kevent_t, bufferSize, bufferSize)
	var timeout = syscall.NsecToTimespec(1e9)

	for {
		select {
		case <-my.wc.C():
			return
		default:
			my.pollData(events, timeout)
		}
	}
}

func (my *Poll) Close() error {
	my.wc.Close(func() {
		my.changes.Lock()
		my.changes.d = nil
		my.connections = loom.Map{}
		my.changes.Unlock()

		_ = syscall.Close(my.fd)
	})

	return nil
}

// 记录当前活跃的链接，出错后通过Remove方法移除
func (my *Poll) add(conn net.Conn) *WSConn {

	var fd = socketFD(conn)
	var event = syscall.Kevent_t{Ident: fd, Flags: syscall.EV_ADD | syscall.EV_EOF, Filter: syscall.EVFILT_READ}
	var receivedChan = make(chan Message, 8)
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

func (my *Poll) remove(item *WSConn) {
	my.changes.Lock()
	{
		// 找到fd出现的位置
		var changes = my.changes.d
		var count = len(my.changes.d)
		var idxFind = -1
		for i := 0; i < count; i++ {
			if changes[i].Ident == item.fd {
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
	_ = item.conn.Close()
}

func (my *Poll) pollData(events []syscall.Kevent_t, timeout syscall.Timespec) {
retry:
	my.changes.RLock()
	num, err := syscall.Kevent(my.fd, my.changes.d, events, &timeout)
	my.changes.RUnlock()

	if err != nil {
		if err == syscall.EINTR {
			goto retry
		}
		return
	}

	for i := 0; i < num; i++ {
		var ident = events[i].Ident
		var item = my.connections.Get1(ident).(*WSConn)
		var conn = item.conn

		// EOF
		if (events[i].Flags & syscall.EV_EOF) == syscall.EV_EOF {
			item.receivedChan <- Message{Err: io.EOF}
			my.remove(item)
			continue
		}

		var data, _, err = wsutil.ReadClientData(conn)
		if err != nil {
			item.receivedChan <- Message{Err: err}
			my.remove(item)
			continue
		}

		if err := checkReceivedMsgBytes(data); err != nil {
			item.receivedChan <- Message{Err: err}
			my.remove(item)
			continue
		}

		item.receivedChan <- Message{Data: data}
	}
}