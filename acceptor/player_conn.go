package acceptor

import "net"

/********************************************************************
created:    2020-09-01
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type PlayerConn interface {
	GetNextMessage() (b []byte, err error)
	net.Conn
}
