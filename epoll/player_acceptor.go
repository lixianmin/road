package epoll

import (
	"fmt"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/logo"
	"github.com/xtaci/gaio"
	"sync/atomic"
)

/********************************************************************
created:    2020-12-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/
type PlayerAcceptor struct {
	watcher  *gaio.Watcher
	isClosed int32
}

func newPlayerAcceptor() *PlayerAcceptor {
	var watcher, err = gaio.NewWatcher()
	if err != nil {
		var message = fmt.Sprintf("watcher is %v, err=%q", watcher, err)
		panic(message)
		return nil
	}

	var my = &PlayerAcceptor{
		watcher: watcher,
	}

	go my.goWatcher(watcher)
	return my
}

func (my *PlayerAcceptor) goWatcher(watcher *gaio.Watcher) {
	defer loom.DumpIfPanic()
	defer watcher.Close()

	for !my.IsClosed() {
		var results, err = watcher.WaitIO()
		if err != nil {
			logo.Warn("err=%q", err)
			return
		}

		for _, item := range results {
			if item.Error != nil {
				if playerConn, ok := item.Context.(PlayerConn); ok {
					playerConn.sendErrorMessage(item.Error)
				}
				continue
			}

			switch item.Operation {
			case gaio.OpRead:
				if playerConn, ok := item.Context.(PlayerConn); ok {
					if item.Size > 0 {
						err = playerConn.onReceiveData(item.Buffer[:item.Size])
						if err != nil {
							playerConn.sendErrorMessage(err)
							_ = watcher.Free(item.Conn)
							continue
						}
					}

					// 每次想接收数据都得使用watcher.Read()重新发起一次调用，在此之前是不能接收到新数据的
					err = watcher.Read(item.Context, item.Conn, nil)
					if err != nil {
						playerConn.sendErrorMessage(err)
						_ = watcher.Free(item.Conn)
					}
				}
			case gaio.OpWrite:
			}
		}
	}
}

func (my *PlayerAcceptor) getWatcher() *gaio.Watcher {
	return my.watcher
}

func (my *PlayerAcceptor) Close() error {
	atomic.StoreInt32(&my.isClosed, 1)
	return nil
}

func (my *PlayerAcceptor) IsClosed() bool {
	return atomic.LoadInt32(&my.isClosed) == 1
}
