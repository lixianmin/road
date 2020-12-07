package epoll

import (
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/logger"
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

	var watcher = my.getWatcher()
	if watcher == nil {
		logger.Warn("watcher is nil")
		return
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Warn("Failed to listen on address=%q, err=%q", address, err)
		return
	}
	defer listener.Close()

	for !my.IsClosed() {
		conn, err := listener.Accept()
		if err != nil {
			logger.Info("Failed to accept TCP connection: %s", err)
			continue
		}

		var item = newTcpConn(conn, watcher, receivedChanSize)
		if item != nil {
			var err = watcher.Read(item, conn, nil)
			if err == nil {
				my.connChan <- item
			}
		}
	}
}

func (my *TcpAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
