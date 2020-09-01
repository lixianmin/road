package logger

import "github.com/lixianmin/logo"

/********************************************************************
created:    2020-04-25
author:     lixianmin

Copyright (C) - All Rights Reserved
 *********************************************************************/

var theLogger logo.ILogger = logo.GetLogger()

func Init(log logo.ILogger) {
	if log != nil {
		theLogger = log
	}
}

func GetLogger() logo.ILogger {
	return theLogger
}

func Debug(first interface{}, args ...interface{}) {
	theLogger.Debug(first, args...)
}

func Info(first interface{}, args ...interface{}) {
	theLogger.Info(first, args...)
}

func Warn(first interface{}, args ...interface{}) {
	theLogger.Warn(first, args...)
}

func Error(first interface{}, args ...interface{}) {
	theLogger.Error(first, args...)
}
