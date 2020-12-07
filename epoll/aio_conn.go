package epoll

import (
	"bytes"
	"github.com/lixianmin/road/conn/codec"
	"github.com/xtaci/gaio"
	"net"
)

/********************************************************************
created:    2020-12-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type AioConn struct {
	conn          net.Conn
	watcher       *gaio.Watcher
	receivedChan  chan Message
	inboundBuffer *bytes.Buffer
}

func newAioConn(conn net.Conn, watcher *gaio.Watcher, receivedChanSize int) *AioConn {
	var receivedChan = make(chan Message, receivedChanSize)
	var my = &AioConn{
		conn:          conn,
		watcher:       watcher,
		receivedChan:  receivedChan,
		inboundBuffer: gBufferPool.Get(),
	}

	return my
}

func (my *AioConn) GetReceivedChan() <-chan Message {
	return my.receivedChan
}

func (my *AioConn) onReceiveData(buff []byte) error {
	var inboundBuffer = my.inboundBuffer
	var _, err = inboundBuffer.Write(buff)
	if err != nil {
		return err
	}

	var headLength = codec.HeadLength
	var data = inboundBuffer.Bytes()

	for len(data) > headLength {
		var header = data[:headLength]
		msgSize, _, err := codec.ParseHeader(header)
		if err != nil {
			return err
		}

		var totalSize = headLength + msgSize
		if len(data) < totalSize {
			return nil
		}

		var frameData = make([]byte, totalSize)
		copy(frameData, data[:totalSize])
		my.receivedChan <- Message{Data: frameData}

		inboundBuffer.Next(totalSize)
		data = inboundBuffer.Bytes()
	}

	// 调整inboundBuffer的offset
	if len(data) == 0 {
		inboundBuffer.Reset()
	} else {
		my.inboundBuffer = gBufferPool.Get()
		my.inboundBuffer.Write(data)

		inboundBuffer.Reset()
		gBufferPool.Put(inboundBuffer)
	}

	return nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (my *AioConn) Write(b []byte) (int, error) {
	return len(b), my.watcher.Write(my, my.conn, b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (my *AioConn) Close() error {
	return my.watcher.Free(my.conn)
}

// LocalAddr returns the local address.
func (my *AioConn) LocalAddr() net.Addr {
	return my.conn.LocalAddr()
}

// RemoteAddr returns the remote address.
func (my *AioConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}
