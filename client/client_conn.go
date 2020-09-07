package client

import (
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"net"
)

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type clientConn struct {
	conn net.Conn
}

func (my *clientConn) Read(b []byte) (int, error) {
	var reader = wsutil.NewReader(my.conn, ws.StateClientSide)
	return reader.Read(b)
}

func (my *clientConn) Write(b []byte) (int, error) {
	var err = wsutil.WriteClientBinary(my.conn, b)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

func (my *clientConn) Close() error {
	return my.conn.Close()
}
