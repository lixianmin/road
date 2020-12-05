package epoll

import (
	"github.com/lixianmin/road/logger"
	"github.com/panjf2000/gnet"
	"sync/atomic"
	"time"
)

/********************************************************************
created:    2020-12-03
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/
type EventAcceptor struct {
	connChan    chan PlayerConn
	options     acceptorOptions
	idGenerator int64
}

func NewEventAcceptor(address string, opts ...AcceptorOption) *EventAcceptor {
	var options = acceptorOptions{
		ConnChanSize:     16,
		ReceivedChanSize: 16,
		PollBufferSize:   1024,
	}

	for _, opt := range opts {
		opt(&options)
	}

	var my = &EventAcceptor{
		connChan: make(chan PlayerConn, options.ConnChanSize),
		options:  options,
	}

	go my.goServe(address)
	return my
}

func (my *EventAcceptor) goServe(address string) {
	var protoAddress = "tcp://" + address
	var err = gnet.Serve(my, protoAddress,
		gnet.WithMulticore(true),
		gnet.WithCodec(newEventCodec()),
		gnet.WithLogger(newEventLogger()))

	if err != nil {
		logger.Warn(err)
	}
}

// OnInitComplete fires when the server is ready for accepting connections.
// The parameter:server has information and various utilities.
func (my *EventAcceptor) OnInitComplete(svr gnet.Server) (action gnet.Action) {
	return
}

// OnShutdown fires when the server is being shut down, it is called right after
// all event-loops and connections are closed.
func (my *EventAcceptor) OnShutdown(svr gnet.Server) {
}

// OnOpened fires when a new connection has been opened.
// The parameter:c has information about the connection such as it's local and remote address.
// Parameter:out is the return value which is going to be sent back to the client.
func (my *EventAcceptor) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	var id = atomic.AddInt64(&my.idGenerator, 1)
	var conn = newEventConn(id, c, my.options.ReceivedChanSize)

	c.SetContext(conn)
	my.connChan <- conn

	return
}

// OnClosed fires when a connection has been closed.
// The parameter:err is the last known connection error.
func (my *EventAcceptor) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	c.SetContext(nil)
	return
}

// PreWrite fires just before any data is written to any client socket, this event function is usually used to
// put some code of logging/counting/reporting or any prepositive operations before writing data to client.
func (my *EventAcceptor) PreWrite() {

}

// React fires when a connection sends the server data.
// Call c.Read() or c.ReadN(n) within the parameter:c to read incoming data from client.
// Parameter:out is the return value which is going to be sent back to the client.
func (my *EventAcceptor) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	if conn, ok := c.Context().(*EventConn); ok {
		// 这里的frame都是经过decode解码的，所以一定符合packet结构要求
		conn.receivedChan <- Message{Data: frame}
	}

	return
}

// Tick fires immediately after the server starts and will fire again
// following the duration specified by the delay return value.
func (my *EventAcceptor) Tick() (delay time.Duration, action gnet.Action) {
	return
}

func (my *EventAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
