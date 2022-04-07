package road

import (
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/logo"
)

/********************************************************************
created:    2021-01-16
author:     lixianmin

为什么要摘出这样一个类出来？
同一个链接的read/write 不能放到同一个goroutine中。write很直接，但read的handler
中写什么代码是不确定的，如果其中调用到conn.Write()，就有可能形成死锁：
1. read的handler想结束返回就需要write成功
2. 但现在sendingChan满了，所以write无法成功，因此read的handler也无法结束
3. 因为read的handler无法结束，就导致同一个goroutine中的sendingChan的数据无法提取
4. sendingChan中的数据无法提取出来，就一直是满的

死锁了

Copyright (C) - All Rights Reserved
*********************************************************************/

type sendingItem struct {
	session *sessionImpl
	data    []byte
}

type sessionSender struct {
	sendingChan chan sendingItem
}

func newSessionSender(chanSize int) *sessionSender {
	var my = &sessionSender{
		sendingChan: make(chan sendingItem, chanSize),
	}

	loom.Go(my.goLoop)
	return my
}

func (my *sessionSender) goLoop(later loom.Later) {
	for {
		select {
		case item := <-my.sendingChan:
			my.onWriteBytes(item.session, item.data)
		}
	}
}

func (my *sessionSender) onWriteBytes(session *sessionImpl, data []byte) {
	select {
	case <-session.wc.C():
	default:
		if _, err := session.conn.Write(data); err != nil {
			logo.Info("close session(%d) by onWriteBytes(), err=%q", session.id, err)
			_ = session.Close()
		}
	}
}
