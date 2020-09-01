package bugfly

import (
	"github.com/gorilla/websocket"
	"github.com/lixianmin/gonsole/logger"
	"net"
	"net/http"
)

/********************************************************************
created:    2020-08-27
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type WSAcceptor struct {
	addr     string
	connChan chan PlayerConn
	listener net.Listener
}

func NewWSAcceptor(addr string) *WSAcceptor {
	acceptor := &WSAcceptor{
		addr:     addr,
		connChan: make(chan PlayerConn),
	}

	return acceptor
}

func (my *WSAcceptor) GetAddr() string {
	if my.listener != nil {
		return my.listener.Addr().String()
	}

	return ""
}

func (my *WSAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}

type connHandler struct {
	upgrader *websocket.Upgrader
	connChan chan PlayerConn
}

func (h *connHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(rw, r, nil)
	if err != nil {
		logger.Info("Upgrade failure, URI=%s, Error=%s", r.RequestURI, err.Error())
		return
	}

	c, err := NewWSConn(conn)
	if err != nil {
		logger.Error("Failed to create new ws connection: %s", err.Error())
		return
	}

	h.connChan <- c
}

func (my *WSAcceptor) ListenAndServe() {
	const IOBufferSize = 4096
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  IOBufferSize,
		WriteBufferSize: IOBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	listener, err := net.Listen("tcp", my.addr)
	if err != nil {
		logger.Error("Failed to listen: %q", err)
	}

	my.listener = listener
	my.serve(&upgrader)
}

func (my *WSAcceptor) serve(upgrader *websocket.Upgrader) {
	defer my.Stop()

	_ = http.Serve(my.listener, &connHandler{
		upgrader: upgrader,
		connChan: my.connChan,
	})
}

// Stop stops the acceptor
func (my *WSAcceptor) Stop() {
	err := my.listener.Close()
	if err != nil {
		logger.Info("Failed to stop: %s", err.Error())
	}
}
