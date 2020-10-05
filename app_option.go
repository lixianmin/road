package road

import (
	"github.com/lixianmin/logo"
	"time"
)

/********************************************************************
created:    2020-09-30
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type appOptions struct {
	HeartbeatTimeout         time.Duration // 心跳超时时间
	DataCompression          bool          // 数据是否压缩
	Logger                   logo.ILogger  // 自定义日志对象，默认只输出到控制台
	SessionSendingChanSize   int           // session的发送缓冲区大小
	SessionTaskQueueSize     int           // session的任务队列长度
	SessionRateLimitBySecond int           // session每秒限流
}

type AppOption func(*appOptions)

func WithHeartbeatTimeout(timeout time.Duration) AppOption {
	return func(options *appOptions) {
		if timeout > 0 {
			options.HeartbeatTimeout = timeout
		}
	}
}

func WithDataCompression(compression bool) AppOption {
	return func(options *appOptions) {
		options.DataCompression = compression
	}
}

func WithLogger(logger logo.ILogger) AppOption {
	return func(options *appOptions) {
		if logger != nil {
			options.Logger = logger
		}
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
