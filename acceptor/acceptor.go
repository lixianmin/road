package acceptor

/********************************************************************
created:    2020-08-27
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type Acceptor interface {
	GetConnChan() chan PlayerConn
	ListenAndServe()
}
