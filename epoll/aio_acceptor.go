package epoll

import (
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/logger"
	"github.com/xtaci/gaio"
	"net"
	"sync/atomic"
)

/********************************************************************
created:    2020-12-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/
type AioAcceptor struct {
	connChan chan PlayerConn
	isClosed int32
}

func NewAioAcceptor(address string, opts ...AcceptorOption) *AioAcceptor {
	var options = acceptorOptions{
		ConnChanSize:     16,
		ReceivedChanSize: 16,
		PollBufferSize:   1024,
	}

	for _, opt := range opts {
		opt(&options)
	}

	var my = &AioAcceptor{
		connChan: make(chan PlayerConn, options.ConnChanSize),
	}

	go my.goListener(address, options.ReceivedChanSize)
	return my
}

func (my *AioAcceptor) goListener(address string, receivedChanSize int) {
	defer loom.DumpIfPanic()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Warn("Failed to listen on address=%q, err=%q", address, err)
		return
	}
	defer listener.Close()

	watcher, err := gaio.NewWatcher()
	if err != nil {
		logger.Warn("Failed to create gaio watcher, err=%q", err)
		return
	}

	defer watcher.Close()
	go my.goWatcher(watcher)

	for !my.IsClosed() {
		conn, err := listener.Accept()
		if err != nil {
			logger.Info("Failed to accept TCP connection: %s", err)
			continue
		}

		var item = newAioConn(conn, watcher, receivedChanSize)
		if item != nil {
			var err = watcher.Read(item, conn, nil)
			if err == nil {
				my.connChan <- item
			}
		}
	}
}

func (my *AioAcceptor) goWatcher(watcher *gaio.Watcher) {
	defer loom.DumpIfPanic()

	for !my.IsClosed() {
		var results, err = watcher.WaitIO()
		if err != nil {
			logger.Warn("err=%q", err)
			return
		}

		for _, item := range results {
			if item.Error != nil {
				logger.Info("item.Error=%q", item.Error)
				continue
			}

			switch item.Operation {
			case gaio.OpRead:
				if playerConn, ok := item.Context.(*AioConn); ok {
					err = playerConn.onReceiveData(item.Buffer[:item.Size])
					if err != nil {
						logger.Info("[playerConn.onReceiveData()] err=%q", err)
						_ = watcher.Free(item.Conn)
						continue
					}

					// 每次想接收数据都得使用watcher.Read()重新发起一次调用，在此之前是不能接收到新数据的
					err = watcher.Read(item.Context, item.Conn, nil)
					if err != nil {
						logger.Info("[watcher.Read()] err=%q", err)
						_ = watcher.Free(item.Conn)
						continue
					}
				}
			case gaio.OpWrite:
			}
		}
	}
}

func (my *AioAcceptor) Close() error {
	atomic.StoreInt32(&my.isClosed, 1)
	return nil
}

func (my *AioAcceptor) IsClosed() bool {
	return atomic.LoadInt32(&my.isClosed) == 1
}

func (my *AioAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
