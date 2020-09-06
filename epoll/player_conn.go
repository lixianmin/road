package epoll

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type PlayerConn interface {
	GetReceivedChan() <-chan Message
	Write(b []byte) (int, error)
}
