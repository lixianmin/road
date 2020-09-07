package epoll

import (
	"github.com/gobwas/ws"
	"net"
)

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WSConn struct {
	conn         net.Conn
	fd           int64
	receivedChan chan Message
}

func (my *WSConn) GetReceivedChan() <-chan Message {
	return my.receivedChan
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (my *WSConn) Write(b []byte) (int, error) {
	// var err = wsutil.WriteServerBinary(my.conn, b)
	// 等价于前面注释掉的代码
	var frame = ws.NewFrame(ws.OpBinary, true, b)
	var err = ws.WriteFrame(my.conn, frame)
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

//// LocalAddr returns the local bugfly address.
//func (my *WSConn) LocalAddr() net.Addr {
//	return my.conn.LocalAddr()
//}

// RemoteAddr returns the remote bugfly address.
func (my *WSConn) RemoteAddr() net.Addr {
	return my.conn.RemoteAddr()
}

//// SetDeadline sets the read and write deadlines associated
//// with the connection. It is equivalent to calling both
//// SetReadDeadline and SetWriteDeadline.
////
//// A deadline is an absolute time after which I/O operations
//// fail with a timeout (see type Error) instead of
//// blocking. The deadline applies to all future and pending
//// I/O, not just the immediately following call to Read or
//// Write. After a deadline has been exceeded, the connection
//// can be refreshed by setting a deadline in the future.
////
//// An idle timeout can be implemented by repeatedly extending
//// the deadline after successful Read or Write calls.
////
//// A zero value for t means I/O operations will not time out.
//func (my *WSConn) SetDeadline(t time.Time) error {
//	if err := my.SetReadDeadline(t); err != nil {
//		return err
//	}
//
//	return my.SetWriteDeadline(t)
//}
//
//// SetReadDeadline sets the deadline for future Read calls
//// and any currently-blocked Read call.
//// A zero value for t means Read will not time out.
//func (my *WSConn) SetReadDeadline(t time.Time) error {
//	return my.conn.SetReadDeadline(t)
//}
//
//// SetWriteDeadline sets the deadline for future Write calls
//// and any currently-blocked Write call.
//// Even if write times out, it may return n > 0, indicating that
//// some of the data was successfully written.
//// A zero value for t means Write will not time out.
//func (my *WSConn) SetWriteDeadline(t time.Time) error {
//	return my.conn.SetWriteDeadline(t)
//}
