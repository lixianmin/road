package main

import (
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/epoll"
	"github.com/lixianmin/road/logger"
	"net/http"
	"strings"
	"time"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func main() {
	logo.GetLogger().SetFilterLevel(logo.LevelDebug)
	listenTcp()
	listenWebSocket()
}

func listenTcp() {
	var accept = epoll.NewTcpAcceptor(":4444")
	var app = road.NewApp(accept)
	var room = &Room{}
	_ = app.Register(room, component.WithName("room"), component.WithNameFunc(strings.ToLower))
	testHook(app)
}

func listenWebSocket() {
	var mux = http.NewServeMux()
	var accept = epoll.NewWsAcceptor(mux, "/")
	var app = road.NewApp(accept)

	var room = &Room{}
	_ = app.Register(room, component.WithName("room"), component.WithNameFunc(strings.ToLower))
	testHook(app)

	var err = http.ListenAndServe(":8888", mux)
	println(err)
}

func testHook(app *road.App) {
	app.AddHook(func(rawMethod func() (interface{}, error)) (i interface{}, e error) {
		var startTime = time.Now()
		var ret, err = rawMethod()
		var delta = time.Since(startTime)
		logger.Info(delta)

		return ret, err
	})

	app.AddHook(func(rawMethod func() (interface{}, error)) (i interface{}, e error) {
		logger.Info("hello")
		var ret, err = rawMethod()
		logger.Info("world")
		return ret, err
	})
}
