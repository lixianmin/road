// +build linux

package epoll

import (
	"github.com/gobwas/ws/wsutil"
	"github.com/lixianmin/got/loom"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

/********************************************************************
created:    2020-09-06
author:     lixianmin

参考:  https://github.com/smallnest/epoller/blob/master/epoll_linux.go

Copyright (C) - All Rights Reserved
*********************************************************************/

type Poll struct {
	receivedChanLen int
	fd              int
	connections     loom.Map
	wc              loom.WaitClose
}

type loopArgs struct {
	events []unix.EpollEvent
}

func newPoll(pollBufferSize int, receivedChanLen int) *Poll {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil
	}

	var poll = &Poll{
		receivedChanLen: receivedChanLen,
		fd:              fd,
	}

	loom.Go(func(later *loom.Later) {
		poll.goLoop(later, pollBufferSize)
	})
	return poll
}

func (my *Poll) goLoop(later *loom.Later, bufferSize int) {
	defer my.Close()
	var args = &loopArgs{
		events: make([]unix.EpollEvent, bufferSize, bufferSize),
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
		my.connections = loom.Map{}
		var err = unix.Close(my.fd)
		return err
	})
}

func (my *Poll) add(conn net.Conn) *WSConn {
	// Extract file descriptor associated with the connection
	fd := socketFD(conn)

	err := unix.EpollCtl(my.fd, syscall.EPOLL_CTL_ADD, int(fd), &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return nil
	}

	var playerConn = &WSConn{
		fd:           fd,
		conn:         conn,
		receivedChan: make(chan Message, my.receivedChanLen),
	}

	my.connections.Put(fd, playerConn)
	return playerConn
}

func (my *Poll) remove(item *WSConn) error {
	my.connections.Remove(item.fd)
	_ = item.Close()
	var err = unix.EpollCtl(my.fd, syscall.EPOLL_CTL_DEL, int(item.fd), nil)
	return err
}

func (my *Poll) pollData(args *loopArgs) {
retry:
	var events = args.events
	n, err := unix.EpollWait(my.fd, events, -1)
	if err != nil {
		if err == unix.EINTR {
			goto retry
		}
		return
	}

	for i := 0; i < n; i++ {
		var fd = int64(events[i].Fd)
		var item = my.connections.Get1(fd).(*WSConn)
		if (events[i].Events & unix.POLLHUP) == unix.POLLHUP {
			_ = my.remove(item)
			continue
		}

		var data, _, err = wsutil.ReadClientData(item.conn)
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
