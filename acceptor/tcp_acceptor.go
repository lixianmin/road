package acceptor

import (
	"github.com/lixianmin/road/logger"
	"net"
)

// Copyright (c) nano Author and TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// TCPAcceptor struct
type TCPAcceptor struct {
	addr     string
	connChan chan PlayerConn
	listener net.Listener
	running  bool
	certFile string
	keyFile  string
}

// NewTCPAcceptor creates a new instance of tcp acceptor
func NewTCPAcceptor(addr string) *TCPAcceptor {
	return &TCPAcceptor{
		addr:     addr,
		connChan: make(chan PlayerConn),
		running:  false,
	}
}

// GetAddr returns the addr the acceptor will listen on
func (my *TCPAcceptor) GetAddr() string {
	if my.listener != nil {
		return my.listener.Addr().String()
	}
	return ""
}

// GetConnChan gets a connection channel
func (my *TCPAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}

// Stop stops the acceptor
func (my *TCPAcceptor) Stop() {
	my.running = false
	_ = my.listener.Close()
}

// ListenAndServe using tcp acceptor
func (my *TCPAcceptor) ListenAndServe() {
	listener, err := net.Listen("tcp", my.addr)
	if err != nil {
		logger.Warn("Failed to listen: %s", err.Error())
	}

	my.listener = listener
	my.running = true
	my.serve()
}

func (my *TCPAcceptor) serve() {
	defer my.Stop()
	for my.running {
		conn, err := my.listener.Accept()
		if err != nil {
			logger.Info("Failed to accept TCP connection: %s", err.Error())
			continue
		}

		my.connChan <- &tcpConn{
			Conn: conn,
		}
	}
}
