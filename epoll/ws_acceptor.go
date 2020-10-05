package epoll

import (
	"github.com/gobwas/ws"
	"net/http"
)

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WSAcceptor struct {
	poll     *WSPoll
	connChan chan PlayerConn
}

func NewWSAcceptor(opts ...AcceptorOption) *WSAcceptor {
	var options = acceptorOptions{
		ConnChanSize:     16,
		ReceivedChanSize: 16,
		PollBufferSize:   1024,
	}

	for _, opt := range opts {
		opt(&options)
	}

	var my = &WSAcceptor{
		poll:     newPoll(options.PollBufferSize, options.ReceivedChanSize),
		connChan: make(chan PlayerConn, options.ConnChanSize),
	}

	return my
}

func (my *WSAcceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		return
	}

	var item = my.poll.add(conn)
	if item != nil {
		my.connChan <- item
	}
}

func (my *WSAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
