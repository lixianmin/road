package epoll

import (
	"bytes"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/xtaci/gaio"
	"net"
)

/********************************************************************
created:    2020-12-07
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WebConn struct {
	conn         net.Conn
	watcher      *gaio.Watcher
	receivedChan chan Message
	input        *bytes.Buffer
	readerWriter *WebReaderWriter
}

func newWebConn(conn net.Conn, watcher *gaio.Watcher, receivedChanSize int) *WebConn {
	var receivedChan = make(chan Message, receivedChanSize)
	var my = &WebConn{
		conn:         conn,
		watcher:      watcher,
		receivedChan: receivedChan,
		input:        gBufferPool.Get(),
		readerWriter: NewWebReaderWriter(conn, watcher),
	}

	return my
}

func (my *WebConn) GetReceivedChan() <-chan Message {
	return my.receivedChan
}

func (my *WebConn) onReceiveData(buff []byte) error {
	var err = my.readerWriter.onReceiveData(buff)
	if err != nil {
		return err
	}

	data, _, err := wsutil.ReadData(my.readerWriter, ws.StateServerSide)
	if err != nil {
		my.receivedChan <- Message{Err: err}
		return err
	}

	if err := checkReceivedMsgBytes(data); err != nil {
		my.receivedChan <- Message{Err: err}
		return err
	}

	my.receivedChan <- Message{Data: data}
	return nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (my *WebConn) Write(b []byte) (int, error) {
	var frame = ws.NewBinaryFrame(b)
	var err = ws.WriteFrame(my.readerWriter, frame)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (my *WebConn) Close() error {
	return my.watcher.Free(my.conn)
}

// LocalAddr returns the local address.
func (my *WebConn) LocalAddr() net.Addr {
	return my.conn.LocalAddr()
}

// RemoteAddr returns the remote address.
func (my *WebConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}
