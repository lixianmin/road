package epoll

import (
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/logger"
	"net"
)

/********************************************************************
created:    2020-10-05
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/
type TcpAcceptor struct {
	poll     *TcpPoll
	connChan chan PlayerConn
	wc       loom.WaitClose
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
		poll:     newTcpPoll(options.PollBufferSize, options.ReceivedChanSize),
		connChan: make(chan PlayerConn, options.ConnChanSize),
	}

	loom.Go(func(later loom.Later) {
		my.goLoop(later, address)
	})

	return my
}

// ListenAndServe using tcp acceptor
func (my *TcpAcceptor) goLoop(later loom.Later, address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Warn("Failed to listen: %s", err)
		return
	}

	defer func() {
		_ = listener.Close()
	}()

	for !my.wc.IsClosed() {
		conn, err := listener.Accept()
		if err != nil {
			logger.Info("Failed to accept TCP connection: %s", err)
			continue
		}

		var item = my.poll.add(conn)
		if item != nil {
			my.connChan <- item
		}
	}
}

func (my *TcpAcceptor) Close() error {
	return my.wc.Close(nil)
}

func (my *TcpAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
