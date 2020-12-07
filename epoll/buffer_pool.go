package epoll

import (
	"github.com/lixianmin/road/core"
	"sync"
)

/********************************************************************
created:    2020-12-07
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

var gBufferPool = NewBufferPool(32768)

type BufferPool struct {
	pool          sync.Pool
	maxBufferSize int
}

func NewBufferPool(maxBufferSize int) *BufferPool {
	var my = &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &core.Buffer{}
			},
		},
		maxBufferSize: maxBufferSize,
	}

	return my
}

func (my *BufferPool) Get() *core.Buffer {
	var item = my.pool.Get()
	return item.(*core.Buffer)
}

func (my *BufferPool) Put(buff *core.Buffer) {
	if buff != nil && buff.Cap() <= my.maxBufferSize {
		buff.Reset()
		my.pool.Put(buff)
	}
}
