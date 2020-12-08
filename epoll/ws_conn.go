package epoll

import (
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/lixianmin/road/conn/codec"
	"github.com/xtaci/gaio"
	"io"
	"net"
)

/********************************************************************
created:    2020-12-07
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WsConn struct {
	conn         net.Conn
	watcher      *gaio.Watcher
	receivedChan chan Message
	readerWriter *WsReaderWriter
}

func newWsConn(conn net.Conn, watcher *gaio.Watcher, receivedChanSize int) *WsConn {
	var receivedChan = make(chan Message, receivedChanSize)
	var my = &WsConn{
		conn:         conn,
		watcher:      watcher,
		receivedChan: receivedChan,
		readerWriter: NewWsReaderWriter(conn, watcher),
	}

	return my
}

func (my *WsConn) sendErrorMessage(err error) {
	if err != nil {
		my.receivedChan <- Message{Err: err}
	}
}

func (my *WsConn) GetReceivedChan() <-chan Message {
	return my.receivedChan
}

func (my *WsConn) onReceiveData(buff []byte) error {
	var input = my.readerWriter.input
	_, _ = input.Write(buff)

	for input.Len() > codec.HeadLength {
		var lastOffsetSet = input.GetOffset()
		data, _, err := wsutil.ReadData(my.readerWriter, ws.StateServerSide)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				input.SetOffset(lastOffsetSet)
				return nil
			}

			my.receivedChan <- Message{Err: err}
			return err
		}

		if err := checkReceivedMsgBytes(data); err != nil {
			input.SetOffset(lastOffsetSet)
			return nil
		}

		my.receivedChan <- Message{Data: data}
	}

	input.Tidy()
	//logger.Info("readerSize=%d, len(buff)=%d", my.readerWriter.ReaderSize(), len(buff))
	return nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (my *WsConn) Write(b []byte) (int, error) {
	var frame = ws.NewBinaryFrame(b)
	var err = ws.WriteFrame(my.readerWriter, frame)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (my *WsConn) Close() error {
	return my.watcher.Free(my.conn)
}

// LocalAddr returns the local address.
func (my *WsConn) LocalAddr() net.Addr {
	return my.conn.LocalAddr()
}

// RemoteAddr returns the remote address.
func (my *WsConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}
