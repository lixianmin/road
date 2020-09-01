package main

import (
	"github.com/lixianmin/bugfly"
	"github.com/lixianmin/bugfly/component"
	"github.com/lixianmin/logo"
	"strings"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func main() {
	logo.GetLogger().SetFilterLevel(logo.LevelDebug)
	var accept = bugfly.NewWSAcceptor(":8880")
	var app = bugfly.NewApp(bugfly.AppArgs{
		Acceptor: accept,
	})

	var room = &Room{}
	app.Register(room, component.WithName("room"), component.WithNameFunc(strings.ToLower))
	accept.ListenAndServe()
}
