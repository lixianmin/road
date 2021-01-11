package road

import (
	"time"
)

/********************************************************************
created:    2020-09-30
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type appOptions struct {
	HeartbeatInterval        time.Duration // 心跳间隔
	DataCompression          bool          // 数据是否压缩
	SessionSendingChanSize   int           // session的发送缓冲区大小
	SessionTaskQueueSize     int           // session的任务队列长度
	SessionRateLimitBySecond int           // session每秒限流
}

type AppOption func(*appOptions)

func WithHeartbeatInterval(interval time.Duration) AppOption {
	return func(options *appOptions) {
		if interval > 0 {
			options.HeartbeatInterval = interval
		}
	}
}

func WithDataCompression(compression bool) AppOption {
	return func(options *appOptions) {
		options.DataCompression = compression
	}
}

func WithSessionSendingChanSize(size int) AppOption {
	return func(options *appOptions) {
		if size > 0 {
			options.SessionSendingChanSize = size
		}
	}
}

func WithSessionTaskQueueSize(size int) AppOption {
	return func(options *appOptions) {
		if size > 0 {
			options.SessionTaskQueueSize = size
		}
	}
}

func WithSessionRateLimitBySecond(limit int) AppOption {
	return func(options *appOptions) {
		if limit > 0 {
			options.SessionRateLimitBySecond = limit
		}
	}
}
