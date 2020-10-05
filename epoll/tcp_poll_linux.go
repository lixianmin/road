// +build linux

package epoll

import (
	"github.com/lixianmin/got/loom"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

/********************************************************************
created:    2020-10-05
author:     lixianmin

参考:  https://github.com/smallnest/epoller/blob/master/epoll_linux.go

Copyright (C) - All Rights Reserved
*********************************************************************/

type TcpPoll struct {
	receivedChanSize int
	fd               int
	connections      loom.Map
	wc               loom.WaitClose
}

type tcpPollFetus struct {
	events []unix.EpollEvent
}

func newTcpPoll(pollBufferSize int, receivedChanSize int) *TcpPoll {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil
	}

	var poll = &TcpPoll{
		receivedChanSize: receivedChanSize,
		fd:               fd,
	}

	loom.Go(func(later loom.Later) {
		poll.goLoop(later, pollBufferSize)
	})
	return poll
}

func (my *TcpPoll) goLoop(later loom.Later, bufferSize int) {
	defer my.Close()
	var fetus = &tcpPollFetus{
		events: make([]unix.EpollEvent, bufferSize, bufferSize),
	}

	var closeChan = my.wc.C()
	for {
		select {
		case <-closeChan:
			return
		default:
			my.pollData(fetus)
		}
	}
}

func (my *TcpPoll) Close() error {
	return my.wc.Close(func() error {
		my.connections = loom.Map{}
		var err = unix.Close(my.fd)
		return err
	})
}

func (my *TcpPoll) add(conn net.Conn) *WsConn {
	// Extract file descriptor associated with the connection
	fd := socketFD(conn)

	err := unix.EpollCtl(my.fd, syscall.EPOLL_CTL_ADD, int(fd), &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return nil
	}

	var playerConn = &WsConn{
		fd:           fd,
		conn:         conn,
		receivedChan: make(chan Message, my.receivedChanSize),
	}

	my.connections.Put(fd, playerConn)
	return playerConn
}

func (my *TcpPoll) remove(item *tcpConn) error {
	my.connections.Remove(item.fd)
	_ = item.Close()
	var err = unix.EpollCtl(my.fd, syscall.EPOLL_CTL_DEL, int(item.fd), nil)
	return err
}

func (my *TcpPoll) pollData(fetus *tcpPollFetus) {
retry:
	var events = fetus.events
	n, err := unix.EpollWait(my.fd, events, -1)
	if err != nil {
		if err == unix.EINTR {
			goto retry
		}
		return
	}

	for i := 0; i < n; i++ {
		var fd = int64(events[i].Fd)
		var item = my.connections.Get1(fd).(*tcpConn)
		if (events[i].Events & unix.POLLHUP) == unix.POLLHUP {
			_ = my.remove(item)
			continue
		}

		var data, err = item.GetNextMessage()
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
