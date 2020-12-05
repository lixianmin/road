package epoll

import (
	"github.com/panjf2000/gnet"
	"net"
)

/********************************************************************
created:    2020-12-03
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type EventConn struct {
	id           int64
	conn         gnet.Conn
	receivedChan chan Message
}

func newEventConn(id int64, conn gnet.Conn, receivedChanSize int) *EventConn {
	var receivedChan = make(chan Message, receivedChanSize)
	var my = &EventConn{
		id:           id,
		conn:         conn,
		receivedChan: receivedChan,
	}

	return my
}

func (my *EventConn) GetReceivedChan() <-chan Message {
	return my.receivedChan
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (my *EventConn) Write(b []byte) (int, error) {
	return len(b), my.conn.AsyncWrite(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (my *EventConn) Close() error {
	return my.conn.Close()
}

// LocalAddr returns the local address.
func (my *EventConn) LocalAddr() net.Addr {
	return my.conn.LocalAddr()
}

// RemoteAddr returns the remote address.
func (my *EventConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}