package epoll

import (
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/logo"
	"net"
)

/********************************************************************
created:    2020-12-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type TcpAcceptor struct {
	*PlayerAcceptor
	connChan chan PlayerConn
}

func NewTcpAcceptor(address string, opts ...AcceptorOption) *TcpAcceptor {
	var options = acceptorOptions{
		ConnChanSize:     16,
		ReceivedChanSize: 16,
		PollBufferSize:   1024,
	}

	for _, opt := range opts {
		opt(&options)
	}

	var my = &TcpAcceptor{
		PlayerAcceptor: newPlayerAcceptor(),
		connChan:       make(chan PlayerConn, options.ConnChanSize),
	}

	go my.goListener(address, options.ReceivedChanSize)
	return my
}

func (my *TcpAcceptor) goListener(address string, receivedChanSize int) {
	defer loom.DumpIfPanic()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logo.Warn("failed to listen on address=%q, err=%q", address, err)
		return
	}
	defer listener.Close()

	var watcher = my.getWatcher()
	for !my.IsClosed() {
		conn, err := listener.Accept()
		if err != nil {
			logo.Info("failed to accept TCP connection: %q", err)
			continue
		}

		var connection = newTcpConn(conn, watcher, receivedChanSize)
		if connection != nil {
			var err = watcher.Read(connection, conn, nil)
			if err == nil {
				my.connChan <- connection
			}
		}
	}
}

func (my *TcpAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
