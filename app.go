package road

import (
	"encoding/json"
	"fmt"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/docgenerator"
	"github.com/lixianmin/road/epoll"
	"github.com/lixianmin/road/logger"
	"github.com/lixianmin/road/route"
	"github.com/lixianmin/road/serialize"
	"github.com/lixianmin/road/util/compression"
	"time"
)

/********************************************************************
created:    2020-08-28
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type (
	HookFunc func(rawMethod func() (interface{}, error)) (interface{}, error)
	App      struct {
		// 下面这组参数，有session里都会用到
		handlers              map[string]*component.Handler // all handler method
		packetEncoder         codec.PacketEncoder
		packetDecoder         codec.PacketDecoder
		messageEncoder        message.Encoder
		serializer            serialize.Serializer
		wheelSecond           *loom.TimingWheel
		heartbeatTimeout      time.Duration
		heartbeatPacketData   []byte
		handshakeResponseData []byte

		accept   *epoll.Acceptor
		sessions loom.Map
		wc       loom.WaitClose
		tasks    *loom.TaskChan

		services     map[string]*component.Service // all registered service
		hookCallback HookFunc
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
		handlers:         make(map[string]*component.Handler, 8),
		packetDecoder:    codec.NewPomeloPacketDecoder(),
		packetEncoder:    codec.NewPomeloPacketEncoder(),
		messageEncoder:   message.NewMessagesEncoder(args.DataCompression),
		serializer:       serialize.NewJsonSerializer(),
		wheelSecond:      loom.NewTimingWheel(time.Second, int(args.HeartbeatTimeout/time.Second)+1),
		heartbeatTimeout: args.HeartbeatTimeout,

		accept:   accept,
		services: make(map[string]*component.Service),
		hookCallback: func(rawMethod func() (interface{}, error)) (i interface{}, e error) {
			return rawMethod()
		},
	}

	app.heartbeatPacketData = app.encodeHeartbeatData()
	app.handshakeResponseData = app.encodeHandshakeData(args.DataCompression)
	app.tasks = loom.NewTaskChan(args.TaskChanSize, app.wc.C())

	loom.Go(app.goLoop)
	return app
}

func (my *App) goLoop(later loom.Later) {
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

func (my *App) onNewSession(args *loopArgsApp, conn epoll.PlayerConn) {
	var session = NewSession(my, conn)

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

func (my *App) Register(comp component.Component, opts ...component.Option) error {
	s := component.NewService(comp, opts)

	if _, ok := my.services[s.Name]; ok {
		return fmt.Errorf("handler: service already defined: %s", s.Name)
	}

	if err := s.ExtractHandler(); err != nil {
		return err
	}

	// register all handlers
	my.services[s.Name] = s
	for name, handler := range s.Handlers {
		var route1 = fmt.Sprintf("%s.%s", s.Name, name)
		my.handlers[route1] = handler
		logger.Debug("route=%s", route1)
	}

	return nil
}

func (my *App) AddHook(callback HookFunc) {
	var last = my.hookCallback
	my.hookCallback = func(rawMethod func() (interface{}, error)) (i interface{}, e error) {
		return callback(func() (i interface{}, e error) {
			return last(rawMethod)
		})
	}
}

func (my *App) getHandler(rt *route.Route) (*component.Handler, error) {
	handler, ok := my.handlers[rt.Short()]
	if !ok {
		e := fmt.Errorf("handler: %s not found", rt.String())
		return nil, e
	}

	return handler, nil
}

// Documentation returns handler and remotes documentacion
func (my *App) Documentation(getPtrNames bool) (map[string]interface{}, error) {
	handlerDocs, err := docgenerator.HandlersDocs("game", my.services, getPtrNames)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"handlers": handlerDocs,}, nil
}

func (my *App) encodeHeartbeatData() []byte {
	var bytes, err = my.packetEncoder.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}

	return bytes
}

func (my *App) encodeHandshakeData(dataCompression bool) []byte {
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

	bytes, err := my.packetEncoder.Encode(packet.Handshake, data)
	if err != nil {
		panic(err)
	}

	return bytes
}
