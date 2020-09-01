package bugfly

import "net"

/********************************************************************
created:    2020-08-27
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type PlayerConn interface {
	GetNextMessage() (b []byte, err error)
	net.Conn
}

type Acceptor interface {
	GetConnChan() chan PlayerConn
}
