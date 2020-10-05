package epoll

import "net/http"

/********************************************************************
created:    2020-06-03
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type IServeMux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}
