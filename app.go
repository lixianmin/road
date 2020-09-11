package road

import (
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/epoll"
	"github.com/lixianmin/road/logger"
	"github.com/lixianmin/road/service"
)

/********************************************************************
created:    2020-08-28
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type (
	App struct {
		commonSessionArgs
		accept   *epoll.Acceptor
		sessions loom.Map
		wc       loom.WaitClose
		tasks    *loom.TaskChan

		handlerService *service.HandlerService
	}

	loopArgsApp struct {
		onHandShakenHandlers []func(*Session)
	}
)

func NewApp(args AppArgs) *App {
	args.checkInit()
	logger.Init(args.Logger)

	var accept = epoll.NewAcceptor(epoll.AcceptorArgs{})
	args.ServeMux.HandleFunc(args.ServePath, accept.ServeHTTP)

	var app = &App{
		commonSessionArgs: *newCommonSessionArgs(args.DataCompression, args.HeartbeatTimeout),
		accept:            accept,
		handlerService:    service.NewHandlerService(),
	}

	app.tasks = loom.NewTaskChan(app.wc.C())
	loom.Go(app.goLoop)
	return app
}

func (my *App) goLoop(later *loom.Later) {
	var args = &loopArgsApp{
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

func (my *App) onNewSession(args *loopArgsApp, conn epoll.PlayerConn) {
	var session = NewSession(conn, my.commonSessionArgs)

	var id = session.Id()
	my.sessions.Put(id, session)

	session.OnClosed(func() {
		my.sessions.Remove(id)
	})

	for _, handler := range args.onHandShakenHandlers {
		session.OnHandShaken(func() {
			handler(session)
		})
	}
}

// 暴露一个OnConnected()事件暂时没有看到很大的意义，因为handshake必须是第一个消息
func (my *App) OnHandShaken(handler func(*Session)) {
	if handler == nil {
		return
	}

	my.tasks.SendCallback(func(args interface{}) (result interface{}, err error) {
		var fetus = args.(*loopArgsApp)
		fetus.onHandShakenHandlers = append(fetus.onHandShakenHandlers, handler)
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
