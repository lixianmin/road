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
	address  string
	poll     *TcpPoll
	connChan chan PlayerConn
	isClosed int32
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
		address:  address,
		poll:     newTcpPoll(options.PollBufferSize, options.ReceivedChanSize),
		connChan: make(chan PlayerConn, options.ConnChanSize),
	}

	loom.Go(my.goLoop)
	return my
}

// ListenAndServe using tcp acceptor
func (my *TcpAcceptor) goLoop(later loom.Later) {
	listener, err := net.Listen("tcp", my.address)
	if err != nil {
		logger.Warn("Failed to listen: %s", err)
		return
	}

	defer func() {
		_ = listener.Close()
	}()

	for !loom.LoadBool(&my.isClosed) {
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
	loom.StoreBool(&my.isClosed, true)
	return nil
}

func (my *TcpAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
