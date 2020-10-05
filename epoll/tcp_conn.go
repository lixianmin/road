package epoll

import (
	"bytes"
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/ifs"
	"io"
	"net"
)

/********************************************************************
created:    2020-10-05
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type TcpConn struct {
	conn         net.Conn
	fd           int64
	receivedChan chan Message
}

func newTcpConn(conn net.Conn, fd int64, receivedChanSize int) *TcpConn {
	var receivedChan = make(chan Message, receivedChanSize)
	var my = &TcpConn{
		conn:         conn,
		fd:           fd,
		receivedChan: receivedChan,
	}

	return my
}

func (my *TcpConn) GetReceivedChan() <-chan Message {
	return my.receivedChan
}

// GetNextMessage reads the next message available in the stream
func (my *TcpConn) GetNextMessage() (b []byte, err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()

	var buff bytes.Buffer
	_, err = buff.ReadFrom(io.LimitReader(my.conn, codec.HeadLength))
	if err != nil {
		return nil, err
	}

	var header = buff.Bytes()
	msgSize, _, err := codec.ParseHeader(header)
	if err != nil {
		return nil, err
	}

	_, err = buff.ReadFrom(io.LimitReader(my.conn, int64(msgSize)))
	if err != nil {
		return nil, err
	}

	var total = buff.Bytes()
	if len(total) < codec.HeadLength+msgSize {
		return nil, ifs.ErrReceivedMsgSmallerThanExpected
	}

	return total, nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (my *TcpConn) Write(b []byte) (int, error) {
	return my.conn.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (my *TcpConn) Close() error {
	return my.conn.Close()
}

// LocalAddr returns the local address.
func (my *TcpConn) LocalAddr() net.Addr {
	return my.conn.LocalAddr()
}

// RemoteAddr returns the remote address.
func (my *TcpConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}
