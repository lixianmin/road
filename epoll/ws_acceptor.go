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

type WsAcceptor struct {
	poll     *WsPoll
	connChan chan PlayerConn
}

func NewWsAcceptor(serveMux IServeMux, servePath string, opts ...AcceptorOption) *WsAcceptor {
	var options = acceptorOptions{
		ConnChanSize:     16,
		ReceivedChanSize: 16,
		PollBufferSize:   1024,
	}

	for _, opt := range opts {
		opt(&options)
	}

	var my = &WsAcceptor{
		poll:     newWsPoll(options.PollBufferSize, options.ReceivedChanSize),
		connChan: make(chan PlayerConn, options.ConnChanSize),
	}

	serveMux.HandleFunc(servePath, my.ServeHTTP)
	return my
}

func (my *WsAcceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (my *WsAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
