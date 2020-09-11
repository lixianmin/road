package road

import (
	"github.com/lixianmin/logo"
	"time"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type AppArgs struct {
	ServeMux         IServeMux
	ServePath        string        // 服务监听的路径
	HeartbeatTimeout time.Duration // 心跳超时时间
	DataCompression  bool          // 数据是否压缩
	Logger           logo.ILogger  // 自定义日志对象，默认只输出到控制台
}

func (args *AppArgs) checkInit() {
	// 最小也得1s
	if args.HeartbeatTimeout < time.Second {
		args.HeartbeatTimeout = 5 * time.Second
	}
}
