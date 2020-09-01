package bugfly

import (
	"encoding/json"
	"github.com/lixianmin/gonsole/logger"
	"github.com/lixianmin/gonsole/bugfly/component"
	"github.com/lixianmin/gonsole/bugfly/conn/codec"
	"github.com/lixianmin/gonsole/bugfly/conn/message"
	"github.com/lixianmin/gonsole/bugfly/conn/packet"
	"github.com/lixianmin/gonsole/bugfly/serialize"
	"github.com/lixianmin/gonsole/bugfly/service"
	"github.com/lixianmin/gonsole/bugfly/util/compression"
	"github.com/lixianmin/got/loom"
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
		acceptor Acceptor
		sessions loom.Map
		wc       loom.WaitClose
		tasks    *loom.TaskChan

		handlerService *service.HandlerService
	}

	AppLoopArgs struct {
		onSessionConnectedCallbacks []func(*Session)
	}
)

func NewApp(args AppArgs) *App {
	checkAppArgs(&args)

	var common = commonSessionArgs{
		packetDecoder:    codec.NewPomeloPacketDecoder(),
		packetEncoder:    codec.NewPomeloPacketEncoder(),
		messageEncoder:   message.NewMessagesEncoder(args.DataCompression),
		serializer:       serialize.NewJsonSerializer(),
		heartbeatTimeout: args.HeartbeatTimeout,
	}

	var app = &App{
		commonSessionArgs: common,
		acceptor:          args.Acceptor,
		handlerService:    service.NewHandlerService(),
	}

	app.tasks = loom.NewTaskChan(app.wc.C())
	app.heartbeatDataEncode(args.DataCompression)
	loom.Go(app.goLoop)
	return app
}

func checkAppArgs(args *AppArgs) {
	if args.HeartbeatTimeout == 0 {
		args.HeartbeatTimeout = 10 * time.Second
	}
}

func (my *App) goLoop(later *loom.Later) {
	var args = &AppLoopArgs{
	}

	for {
		select {
		case conn := <-my.acceptor.GetConnChan():
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

func (my *App) onNewSession(fetus *AppLoopArgs, conn PlayerConn) {
	var session = NewSession(conn, my.commonSessionArgs)

	var id = session.GetSessionId()
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

		for _, callback := range fetus.onSessionConnectedCallbacks {
			callback(session)
		}
	}
}

func (my *App) OnSessionConnected(callback func(*Session)) {
	if callback == nil {
		return
	}

	my.tasks.SendCallback(func(args interface{}) (result interface{}, err error) {
		var fetus = args.(*AppLoopArgs)
		fetus.onSessionConnectedCallbacks = append(fetus.onSessionConnectedCallbacks, callback)
		return nil, nil
	})
}

func (my *App) heartbeatDataEncode(dataCompression bool) {
	hData := map[string]interface{}{
		"code": 200,
		"sys": map[string]interface{}{
			"heartbeat":  my.heartbeatTimeout.Seconds(),
			"dict":       message.GetDictionary(),
			"serializer": my.serializer.GetName(),
		},
	}

	data, err := json.Marshal(hData)
	if err != nil {
		panic(err)
	}

	if dataCompression {
		compressedData, err := compression.DeflateData(data)
		if err != nil {
			panic(err)
		}

		if len(compressedData) < len(data) {
			data = compressedData
		}
	}

	my.handshakeResponseData, err = my.packetEncoder.Encode(packet.Handshake, data)
	if err != nil {
		panic(err)
	}

	my.heartbeatPacketData, err = my.packetEncoder.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}
}
