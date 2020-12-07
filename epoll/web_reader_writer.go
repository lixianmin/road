package epoll

import (
	"bytes"
	"github.com/xtaci/gaio"
	"net"
)

/********************************************************************
created:    2020-12-07
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WebReaderWriter struct {
	conn    net.Conn
	watcher *gaio.Watcher
	input   *bytes.Buffer
}

func NewWebReaderWriter(conn net.Conn, watcher *gaio.Watcher) *WebReaderWriter {
	var my = &WebReaderWriter{
		conn:    conn,
		watcher: watcher,
		input:   gBufferPool.Get(),
	}

	return my
}

func (my *WebReaderWriter) onReceiveData(buff []byte) error {
	var _, err = my.input.Write(buff)
	return err
}

func (my *WebReaderWriter) Read(p []byte) (n int, err error) {
	var input = my.input
	n, err = input.Read(p)

	var remain = input.Bytes()
	if len(remain) == 0 {
		input.Reset()
	} else {
		my.input = gBufferPool.Get()
		my.input.Write(remain)

		input.Reset()
		gBufferPool.Put(input)
	}

	return n, err
}

func (my *WebReaderWriter) Write(p []byte) (n int, err error) {
	return len(p), my.watcher.Write(my, my.conn, p)
}
