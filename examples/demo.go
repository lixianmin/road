package main

import (
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/logger"
	"net/http"
	"reflect"
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
	var mux = http.NewServeMux()
	var app = road.NewApp(road.AppArgs{
		ServeMux:  mux,
		ServePath: "/",
	})

	var room = &Room{}
	_ = app.Register(room, component.WithName("room"), component.WithNameFunc(strings.ToLower))
	testHook(app)

	var err = http.ListenAndServe(":8888", nil)
	println(err)
}

func testHook(app *road.App) {
	app.AddHook(func(rawMethod road.RawMethod, method reflect.Method, args []reflect.Value) (i interface{}, e error) {
		var startTime = time.Now()
		var ret, err = rawMethod(method, args)
		var delta = time.Since(startTime)
		logger.Info(delta)

		return ret, err
	})

	app.AddHook(func(rawMethod road.RawMethod, method reflect.Method, args []reflect.Value) (i interface{}, e error) {
		logger.Info("hello")
		var ret, err = rawMethod(method, args)
		logger.Info("world")
		return ret, err
	})
}
