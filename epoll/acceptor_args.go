package epoll

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type AcceptorArgs struct {
	ConnChanLen     int // GetConnChan()返回
	ReceivedChanLen int // 每一个PlayerConn拥有一个receivedChan
	PollBufferSize  int // poll的时间缓冲的长度
}

func checkAcceptorArgs(args *AcceptorArgs) {
	if args.ConnChanLen <= 0 {
		args.ConnChanLen = 16
	}

	if args.ReceivedChanLen <= 0 {
		args.ReceivedChanLen = 8
	}

	if args.PollBufferSize <= 0 {
		args.PollBufferSize = 1024
	}
}
