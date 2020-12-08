package epoll

import "net"

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type PlayerConn interface {
	onReceiveData(buff []byte) error
	sendErrorMessage(err error)

	GetReceivedChan() <-chan Message
	Write(b []byte) (int, error)
	Close() error
	RemoteAddr() net.Addr
}
