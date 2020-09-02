package bugfly

import (
	"context"
	"github.com/lixianmin/road/ifs"
	"github.com/lixianmin/road/logger"
	"time"
)

/********************************************************************
created:    2020-09-01
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func GetSessionFromCtx(ctx context.Context) *Session {
	fetus := ctx.Value(ifs.CtxKeySession)
	if fetus == nil {
		logger.Warn("ctx doesn't contain the session")
		return nil
	}

	return fetus.(*Session)
}

func GetBeginTimeFromCtx(ctx context.Context) time.Time {
	fetus := ctx.Value(ifs.CtxKeyBeginTime)
	if fetus == nil {
		logger.Warn("ctx doesn't contain beginTime")
		return time.Time{}
	}

	return fetus.(time.Time)
}
