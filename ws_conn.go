package bugfly

import (
	"github.com/gorilla/websocket"
	"github.com/lixianmin/gonsole/bugfly/conn/codec"
	"github.com/lixianmin/gonsole/bugfly/conn/packet"
	"github.com/lixianmin/gonsole/bugfly/constants"
	"io"
	"net"
	"time"
)

/********************************************************************
created:    2020-08-27
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WSConn struct {
	conn   *websocket.Conn
	typ    int // message type
	reader io.Reader
}

// NewWSConn return an initialized *WSConn
func NewWSConn(conn *websocket.Conn) (*WSConn, error) {
	c := &WSConn{conn: conn}

	return c, nil
}

// GetNextMessage reads the next message available in the stream
func (my *WSConn) GetNextMessage() (b []byte, err error) {
	_, msgBytes, err := my.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if len(msgBytes) < codec.HeadLength {
		return nil, packet.ErrInvalidPomeloHeader
	}
	header := msgBytes[:codec.HeadLength]
	msgSize, _, err := codec.ParseHeader(header)
	if err != nil {
		return nil, err
	}
	dataLen := len(msgBytes[codec.HeadLength:])
	if dataLen < msgSize {
		return nil, constants.ErrReceivedMsgSmallerThanExpected
	} else if dataLen > msgSize {
		return nil, constants.ErrReceivedMsgBiggerThanExpected
	}
	return msgBytes, err
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (my *WSConn) Read(b []byte) (int, error) {
	if my.reader == nil {
		t, r, err := my.conn.NextReader()
		if err != nil {
			return 0, err
		}
		my.typ = t
		my.reader = r
	}
	n, err := my.reader.Read(b)
	if err != nil && err != io.EOF {
		return n, err
	} else if err == io.EOF {
		_, r, err := my.conn.NextReader()
		if err != nil {
			return 0, err
		}
		my.reader = r
	}

	return n, nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (my *WSConn) Write(b []byte) (int, error) {
	err := my.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (my *WSConn) Close() error {
	return my.conn.Close()
}

// LocalAddr returns the local bugfly address.
func (my *WSConn) LocalAddr() net.Addr {
	return my.conn.LocalAddr()
}

// RemoteAddr returns the remote bugfly address.
func (my *WSConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (my *WSConn) SetDeadline(t time.Time) error {
	if err := my.SetReadDeadline(t); err != nil {
		return err
	}

	return my.SetWriteDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (my *WSConn) SetReadDeadline(t time.Time) error {
	return my.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (my *WSConn) SetWriteDeadline(t time.Time) error {
	return my.conn.SetWriteDeadline(t)
}
