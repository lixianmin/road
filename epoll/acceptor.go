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

type Acceptor struct {
	poll     *Poll
	connChan chan PlayerConn
}

func NewAcceptor(args AcceptorArgs) *Acceptor {
	checkAcceptorArgs(&args)
	var my = &Acceptor{
		poll:     newPoll(args.PollBufferSize, args.ReceivedChanLen),
		connChan: make(chan PlayerConn, args.ConnChanLen),
	}

	return my
}

func (my *Acceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		return
	}

	my.connChan <- my.poll.add(conn)
}

func (my *Acceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
