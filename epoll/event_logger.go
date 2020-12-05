package epoll

import (
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road/logger"
)

/********************************************************************
created:    2020-12-05
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type EventLogger struct {
	theLogger logo.ILogger
}

func newEventLogger() *EventLogger {
	var my = &EventLogger{
		theLogger: logger.GetLogger(),
	}

	return my
}

func (my *EventLogger) Debugf(format string, args ...interface{}) {
	my.theLogger.Debug(format, args...)
}

func (my *EventLogger) Infof(format string, args ...interface{}) {
	my.theLogger.Info(format, args...)
}

func (my *EventLogger) Warnf(format string, args ...interface{}) {
	my.theLogger.Warn(format, args...)
}

func (my *EventLogger) Errorf(format string, args ...interface{}) {
	my.theLogger.Error(format, args...)
}

func (my *EventLogger) Fatalf(format string, args ...interface{}) {
	my.theLogger.Error(format, args...)
}
