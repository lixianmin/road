package epoll

import "net"

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type PlayerConn interface {
	net.Conn
	GetReceivedChan() <-chan Message
}
