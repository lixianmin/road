package epoll

import (
	"bytes"
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
				return &bytes.Buffer{}
			},
		},
		maxBufferSize: maxBufferSize,
	}

	return my
}

func (my *BufferPool) Get() *bytes.Buffer {
	var item = my.pool.Get()
	return item.(*bytes.Buffer)
}

func (my *BufferPool) Put(buff *bytes.Buffer) {
	if buff != nil && buff.Cap() <= my.maxBufferSize {
		my.pool.Put(buff)
	}
}
