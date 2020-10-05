package epoll

import (
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/ifs"
	"io"
	"io/ioutil"
	"net"
)

/********************************************************************
created:    2020-10-05
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type tcpConn struct {
	conn         net.Conn
	fd           int64
	receivedChan chan Message
}

func (my *tcpConn) GetReceivedChan() <-chan Message {
	return my.receivedChan
}

// GetNextMessage reads the next message available in the stream
func (my *tcpConn) GetNextMessage() (b []byte, err error) {
	header, err := ioutil.ReadAll(io.LimitReader(my.conn, codec.HeadLength))
	if err != nil {
		return nil, err
	}

	msgSize, _, err := codec.ParseHeader(header)
	if err != nil {
		return nil, err
	}

	msgData, err := ioutil.ReadAll(io.LimitReader(my.conn, int64(msgSize)))
	if err != nil {
		return nil, err
	}

	if len(msgData) < msgSize {
		return nil, ifs.ErrReceivedMsgSmallerThanExpected
	}

	return append(header, msgData...), nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (my *tcpConn) Write(b []byte) (int, error) {
	return my.conn.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (my *tcpConn) Close() error {
	return my.conn.Close()
}

// LocalAddr returns the local address.
func (my *tcpConn) LocalAddr() net.Addr {
	return my.conn.LocalAddr()
}

// RemoteAddr returns the remote address.
func (my *tcpConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}
