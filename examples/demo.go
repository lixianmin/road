package main

import (
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road"
	"github.com/lixianmin/road/component"
	"net/http"
	"strings"
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
	app.Register(room, component.WithName("room"), component.WithNameFunc(strings.ToLower))

	_ = http.ListenAndServe(":8888", nil)
}
