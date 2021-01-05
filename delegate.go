package road

import (
	"github.com/lixianmin/logo"
	"sync"
)

/********************************************************************
created:    2020-09-04
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type delegate struct {
	handlers []func()
	lock     sync.Mutex
}

func (my *delegate) Add(handler func()) {
	if handler != nil {
		my.lock.Lock()
		my.handlers = append(my.handlers, handler)
		my.lock.Unlock()
	}
}

func (my *delegate) Invoke() {
	// 单独clone一份出来，因为callback的方法体调用了哪些内容未知，防止循环调用导致死循环
	my.lock.Lock()
	var cloned = make([]func(), len(my.handlers))
	for i, handler := range my.handlers {
		cloned[i] = handler
	}
	my.lock.Unlock()

	defer func() {
		if r := recover(); r != nil {
			logo.Info("[Invoke()] panic: r=%v", r)
		}
	}()

	for _, handler := range cloned {
		handler()
	}
}
