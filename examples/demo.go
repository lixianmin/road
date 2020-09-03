package main

import (
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road"
	"github.com/lixianmin/road/acceptor"
	"github.com/lixianmin/road/component"
	"strings"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func main() {
	logo.GetLogger().SetFilterLevel(logo.LevelDebug)
	var accept = acceptor.NewTCPAcceptor(":8880")
	var app = road.NewApp(road.AppArgs{
		Acceptor: accept,
	})

	var room = &Room{}
	app.Register(room, component.WithName("room"), component.WithNameFunc(strings.ToLower))
	accept.ListenAndServe()
}
