package road

import (
	"encoding/json"
	"fmt"
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/docgenerator"
	"github.com/lixianmin/road/epoll"
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
		wheelSecond           *loom.Wheel
		heartbeatInterval     time.Duration
		heartbeatPacketData   []byte
		handshakeResponseData []byte
		sendingChanSize       int
		taskQueueSize         int
		rateLimitBySecond     int32

		accept   epoll.Acceptor
		sessions loom.Map
		senders  []*sessionSender
		tasks    *loom.TaskQueue
		wc       loom.WaitClose

		services     map[string]*component.Service // all registered service
		hookCallback HookFunc
	}

	appFetus struct {
		onHandShakenHandlers []func(session Session)
	}
)

func NewApp(accept epoll.Acceptor, opts ...AppOption) *App {
	// 默认值
	var options = appOptions{
		HeartbeatInterval:        5 * time.Second,
		DataCompression:          false,
		SenderBufferSize:         4096,
		SenderCount:              16,
		SessionRateLimitBySecond: 2,
	}

	// 初始化
	for _, opt := range opts {
		opt(&options)
	}

	var app = &App{
		handlers:          make(map[string]*component.Handler, 8),
		packetDecoder:     codec.NewPomeloPacketDecoder(),
		packetEncoder:     codec.NewPomeloPacketEncoder(),
		messageEncoder:    message.NewMessagesEncoder(options.DataCompression),
		serializer:        serialize.NewJsonSerializer(),
		wheelSecond:       loom.NewWheel(time.Second, int(options.HeartbeatInterval/time.Second)+1),
		heartbeatInterval: options.HeartbeatInterval,
		sendingChanSize:   options.SenderBufferSize,
		rateLimitBySecond: int32(options.SessionRateLimitBySecond),

		accept:   accept,
		services: make(map[string]*component.Service),
		hookCallback: func(rawMethod func() (interface{}, error)) (i interface{}, e error) {
			return rawMethod()
		},
	}

	app.senders = createSenders(options)
	app.heartbeatPacketData = app.encodeHeartbeatData()
	app.handshakeResponseData = app.encodeHandshakeData(options.DataCompression)

	// 这个tasks，只是内部用一下，不公开
	app.tasks = loom.NewTaskQueue(loom.WithSize(2), loom.WithCloseChan(app.wc.C()))

	loom.Go(app.goLoop)
	return app
}

func (my *App) goLoop(later loom.Later) {
	var fetus = &appFetus{}

	var closeChan = my.wc.C()
	for {
		select {
		case conn := <-my.accept.GetConnChan():
			my.onNewSession(fetus, conn)
		case task := <-my.tasks.C:
			var err = task.Do(fetus)
			if err != nil {
				logo.Info("err=%q", err)
			}
		case <-closeChan:
			return
		}
	}
}

func (my *App) onNewSession(fetus *appFetus, conn epoll.PlayerConn) {
	var session = NewSession(my, conn)

	var id = session.Id()
	my.sessions.Put(id, session)

	session.OnClosed(func() {
		my.sessions.Remove(id)
	})

	// for循环中小心closure的问题
	var handlers = fetus.onHandShakenHandlers
	for i := range handlers {
		var handler = handlers[i]
		session.OnHandShaken(func() {
			handler(session)
		})
	}
}

// OnHandShaken 暴露一个OnConnected()事件暂时没有看到很大的意义，因为handshake必须是第一个消息
func (my *App) OnHandShaken(handler func(session Session)) {
	if handler == nil {
		return
	}

	my.tasks.SendCallback(func(args interface{}) (result interface{}, err error) {
		var fetus = args.(*appFetus)
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
		logo.Debug("route=%s", route1)
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

	return map[string]interface{}{"handlers": handlerDocs}, nil
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
			"heartbeat":  my.heartbeatInterval.Seconds(),
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

func (my *App) getSender(sessionId int64) *sessionSender {
	var index = int(sessionId) % len(my.senders)
	var sender = my.senders[index]
	return sender
}

func createSenders(options appOptions) []*sessionSender {
	var senders = make([]*sessionSender, options.SenderCount)
	for i := 0; i < options.SenderCount; i++ {
		senders[i] = newSessionSender(options.SenderBufferSize)
	}

	return senders
}
