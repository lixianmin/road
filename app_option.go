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
	ServePath        string        // 服务监听的路径
	HeartbeatTimeout time.Duration // 心跳超时时间
	DataCompression  bool          // 数据是否压缩
	Logger           logo.ILogger  // 自定义日志对象，默认只输出到控制台
	TaskChanSize     int           // 任务队列长度
}

type AppOption func(*appOptions)

func WithServePath(path string) AppOption {
	return func(options *appOptions) {
		if path != "" {
			options.ServePath = path
		}
	}
}

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

func WithLogger(log logo.ILogger) AppOption {
	return func(options *appOptions) {
		if log != nil {
			options.Logger = log
		}
	}
}

func WithTaskChanSize(size int) AppOption {
	return func(options *appOptions) {
		if size > 0 {
			options.TaskChanSize = size
		}
	}
}
