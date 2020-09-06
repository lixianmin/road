package epoll

import (
	"github.com/gobwas/ws/wsutil"
	"net"
)

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WSConn struct {
	fd           uint64
	conn         net.Conn
	receivedChan chan Message
}

func (my *WSConn) GetReceivedChan() <-chan Message {
	return my.receivedChan
}

func (my *WSConn) Write(b []byte) (int, error) {
	var err = wsutil.WriteServerBinary(my.conn, b)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

func (my *WSConn) Close() error {
	return my.conn.Close()
}

// RemoteAddr returns the remote bugfly address.
func (my *WSConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}
