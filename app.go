package road

import (
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/acceptor"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/logger"
	"github.com/lixianmin/road/service"
	"time"
)

/********************************************************************
created:    2020-08-28
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type (
	App struct {
		commonSessionArgs
		accept   acceptor.Acceptor
		sessions loom.Map
		wc       loom.WaitClose
		tasks    *loom.TaskChan

		handlerService *service.HandlerService
	}

	appLoopArgs struct {
		onConnectedHandlers []func(*Session)
	}
)

func NewApp(args AppArgs) *App {
	checkAppArgs(&args)
	logger.Init(args.Logger)

	var app = &App{
		commonSessionArgs: *newCommonSessionArgs(args.DataCompression, args.HeartbeatTimeout),
		accept:            args.Acceptor,
		handlerService:    service.NewHandlerService(),
	}

	app.tasks = loom.NewTaskChan(app.wc.C())
	loom.Go(app.goLoop)
	return app
}

func checkAppArgs(args *AppArgs) {
	if args.HeartbeatTimeout == 0 {
		args.HeartbeatTimeout = 10 * time.Second
	}
}

func (my *App) goLoop(later *loom.Later) {
	var args = &appLoopArgs{
	}

	for {
		select {
		case conn := <-my.accept.GetConnChan():
			my.onNewSession(args, conn)
		case task := <-my.tasks.C:
			var err = task.Do(args)
			if err != nil {
				logger.Info("err=%q", err)
			}
		case <-my.wc.C():
			return
		}
	}
}

func (my *App) Register(c component.Component, options ...component.Option) {
	var err = my.handlerService.Register(c, options)
	if err != nil {
		logger.Warn("Failed to register handler: %s", err.Error())
	}
}

func (my *App) onNewSession(fetus *appLoopArgs, conn acceptor.PlayerConn) {
	var session = NewSession(conn, my.commonSessionArgs)

	var id = session.Id()
	my.sessions.Put(id, session)

	session.OnClosed(func() {
		my.sessions.Remove(id)
	})

	{
		defer func() {
			if r := recover(); r != nil {
				if r := recover(); r != nil {
					logger.Info("[onNewSession()] panic: r=%v", r)
				}
			}
		}()

		for _, handler := range fetus.onConnectedHandlers {
			handler(session)
		}
	}
}

func (my *App) OnConnected(handler func(*Session)) {
	if handler == nil {
		return
	}

	my.tasks.SendCallback(func(args interface{}) (result interface{}, err error) {
		var fetus = args.(*appLoopArgs)
		fetus.onConnectedHandlers = append(fetus.onConnectedHandlers, handler)
		return nil, nil
	})
}

// Documentation returns handler and remotes documentacion
func (my *App) Documentation(getPtrNames bool) (map[string]interface{}, error) {
	handlerDocs, err := my.handlerService.Docs(getPtrNames)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"handlers": handlerDocs,
	}, nil
}
