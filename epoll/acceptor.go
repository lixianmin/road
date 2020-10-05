package epoll

/********************************************************************
created:    2020-10-05
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/
type Acceptor interface {
	GetConnChan() chan PlayerConn
}
