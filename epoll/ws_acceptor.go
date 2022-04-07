package epoll

import (
	"github.com/gobwas/ws"
	"net/http"
)

/********************************************************************
created:    2020-12-07
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WsAcceptor struct {
	*PlayerAcceptor
	connChan         chan PlayerConn
	receivedChanSize int
	isClosed         int32
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
		PlayerAcceptor:   newPlayerAcceptor(),
		connChan:         make(chan PlayerConn, options.ConnChanSize),
		receivedChanSize: options.ReceivedChanSize,
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

	var watcher = my.getWatcher()
	if watcher == nil {
		return
	}

	var item = newWsConn(conn, watcher, my.receivedChanSize)
	if item != nil {
		var err = watcher.Read(item, conn, nil)
		if err == nil {
			my.connChan <- item
		}
	}
}

func (my *WsAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
