package epoll

import (
	"github.com/gobwas/ws"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/logger"
	"github.com/xtaci/gaio"
	"net/http"
	"sync/atomic"
)

/********************************************************************
created:    2020-12-07
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/
type WebAcceptor struct {
	connChan         chan PlayerConn
	watcher          *gaio.Watcher
	receivedChanSize int
	isClosed         int32
}

func NewWebAcceptor(serveMux IServeMux, servePath string, opts ...AcceptorOption) *WebAcceptor {
	var options = acceptorOptions{
		ConnChanSize:     16,
		ReceivedChanSize: 16,
		PollBufferSize:   1024,
	}

	for _, opt := range opts {
		opt(&options)
	}

	var watcher, err = gaio.NewWatcher()
	if err != nil {
		logger.Warn("Failed to create gaio watcher, err=%q", err)
		return nil
	}

	var my = &WebAcceptor{
		connChan:         make(chan PlayerConn, options.ConnChanSize),
		watcher:          watcher,
		receivedChanSize: options.ReceivedChanSize,
	}

	go my.goWatcher(watcher)
	serveMux.HandleFunc(servePath, my.ServeHTTP)
	return my
}

func (my *WebAcceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		return
	}

	var item = newWebConn(conn, my.watcher, my.receivedChanSize)
	if item != nil {
		var err = my.watcher.Read(item, conn, nil)
		if err == nil {
			my.connChan <- item
		}
	}
}

func (my *WebAcceptor) goWatcher(watcher *gaio.Watcher) {
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
				if playerConn, ok := item.Context.(*WebConn); ok {
					if item.Size > 0 {
						err = playerConn.onReceiveData(item.Buffer[:item.Size])
						if err != nil {
							logger.Info("[playerConn.onReceiveData()] err=%q", err)
							_ = watcher.Free(item.Conn)
							continue
						}
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

func (my *WebAcceptor) Close() error {
	atomic.StoreInt32(&my.isClosed, 1)
	return nil
}

func (my *WebAcceptor) IsClosed() bool {
	return atomic.LoadInt32(&my.isClosed) == 1
}

func (my *WebAcceptor) GetConnChan() chan PlayerConn {
	return my.connChan
}
