package epoll

import (
	"github.com/xtaci/gaio"
	"net"
)

/********************************************************************
created:    2020-12-07
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WsReaderWriter struct {
	conn    net.Conn
	watcher *gaio.Watcher
	input   *Buffer
}

func NewWsReaderWriter(conn net.Conn, watcher *gaio.Watcher) *WsReaderWriter {
	var my = &WsReaderWriter{
		conn:    conn,
		watcher: watcher,
		input:   &Buffer{},
	}

	return my
}

func (my *WsReaderWriter) Read(p []byte) (n int, err error) {
	n, err = my.input.Read(p)
	return n, err
}

func (my *WsReaderWriter) Write(p []byte) (n int, err error) {
	return len(p), my.watcher.Write(my, my.conn, p)
}
