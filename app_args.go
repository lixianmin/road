package bugfly

import (
	"github.com/lixianmin/bugfly/acceptor"
	"github.com/lixianmin/logo"
	"time"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type AppArgs struct {
	Acceptor         acceptor.Acceptor
	HeartbeatTimeout time.Duration // 心跳超时时间
	DataCompression  bool          // 数据是否压缩
	Logger           logo.ILogger  // 自定义日志对象，默认只输出到控制台
}
