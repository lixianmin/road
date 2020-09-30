package epoll

/********************************************************************
created:    2020-09-30
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type acceptorOptions struct {
	ConnChanSize     int // GetConnChan()返回
	ReceivedChanSize int // 每一个PlayerConn拥有一个receivedChan
	PollBufferSize   int // poll的事件缓冲的长度
}

type AcceptorOption func(*acceptorOptions)

func WithConnChanSize(size int) AcceptorOption {
	return func(options *acceptorOptions) {
		if size > 0 {
			options.ConnChanSize = size
		}
	}
}

func WithReceivedChanSize(size int) AcceptorOption {
	return func(options *acceptorOptions) {
		if size > 0 {
			options.ReceivedChanSize = size
		}
	}
}

func WithPollBufferSize(size int) AcceptorOption {
	return func(options *acceptorOptions) {
		if size > 0 {
			options.PollBufferSize = size
		}
	}
}
