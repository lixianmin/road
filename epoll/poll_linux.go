// +build linux

package epoll

import (
	"net"
	"sync"
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
	fd          int
	connections map[int]net.Conn
	lock        *sync.RWMutex
	connbuf     []net.Conn
	events      []unix.EpollEvent
}

func NewPoll(bufferSize int) (*Poll, error) {
	if bufferSize <= 0 {
		bufferSize = 128
	}

	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &Poll{
		fd:          fd,
		lock:        &sync.RWMutex{},
		connections: make(map[int]net.Conn),
		connbuf:     make([]net.Conn, bufferSize, bufferSize),
		events:      make([]unix.EpollEvent, bufferSize, bufferSize),
	}, nil
}

func (e *Poll) Close() error {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.connections = nil
	return unix.Close(e.fd)
}

func (e *Poll) Add(conn net.Conn) error {
	// Extract file descriptor associated with the connection
	fd := socketFD(conn)

	e.lock.Lock()
	defer e.lock.Unlock()

	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	e.connections[fd] = conn
	return nil
}

func (e *Poll) Remove(conn net.Conn) error {
	fd := socketFD(conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.connections, fd)
	return nil
}

func (e *Poll) Wait() ([]net.Conn, error) {
retry:
	n, err := unix.EpollWait(e.fd, e.events, -1)
	if err != nil {
		if err == unix.EINTR {
			goto retry
		}
		return nil, err
	}

	var connections = e.connbuf[:0]
	e.lock.RLock()
	for i := 0; i < n; i++ {
		conn := e.connections[int(e.events[i].Fd)]
		if (e.events[i].Events & unix.POLLHUP) == unix.POLLHUP {
			_ = conn.Close()
		}

		connections = append(connections, conn)
	}
	e.lock.RUnlock()

	return connections, nil
}
