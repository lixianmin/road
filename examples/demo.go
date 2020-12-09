package main

import (
	"github.com/lixianmin/got/convert"
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road"
	"github.com/lixianmin/road/client"
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

var text = `
created:    2020-08-31
author:     lixianmin

created:    2020-08-31
author:     lixianmin

created:    2020-08-31
author:     lixianmin

created:    2020-08-31
author:     lixianmin

created:    2020-08-31
author:     lixianmin

created:    2020-08-31
author:     lixianmin

created:    2020-08-31
author:     lixianmin

created:    2020-08-31
author:     lixianmin

created:    2020-08-31
author:     lixianmin

`

func main() {
	logo.GetLogger().SetFilterLevel(logo.LevelDebug)
	listenTcp()
	listenWebSocket()

	select {}
}

func listenTcp() {
	var address = ":4444"
	var accept = epoll.NewTcpAcceptor(address, epoll.WithReceivedChanSize(1))
	var app = road.NewApp(accept,
		road.WithSessionRateLimitBySecond(123456789),
		road.WithHeartbeatInterval(2*time.Second),
		road.WithSessionSendingChanSize(1))

	var room = &Room{}
	_ = app.Register(room, component.WithName("room"), component.WithNameFunc(strings.ToLower))
	//testHook(app)

	app.OnHandShaken(func(session *road.Session) {
		logger.Info("session.id=%d", session.Id())
		go func() {
			time.Sleep(5 * time.Second)
			_ = session.Kick()
		}()
	})

	var pClient = client.New()
	if err := pClient.ConnectTo(address); err != nil {
		logger.Error(err.Error())
	}

	go func() {
		time.Sleep(1 * time.Second)

		for i := 0; i < 1000; i++ {
			var item = Enter{Name: "panda", ID: i, Text: text}
			var data = convert.ToJson(item)
			_, err := pClient.SendRequest("room.enter", data)
			if err != nil {
				logger.Error(err.Error())
			}

			_, err2 := pClient.SendRequest("room.sayerror", data)
			if err2 != nil {
				logger.Error(err2)
			}
		}
	}()

	go func() {
		for {
			select {
			case msg := <-pClient.MsgChannel():
				if msg != nil {
					if msg.Err {
						logger.Warn("id=%d, err=%s", msg.Id, string(msg.Data))
					} else {
						var item Enter
						convert.FromJson(msg.Data, &item)
						logger.Info("id=%d, name=%s", item.ID, item.Name)
					}
				}
				break
			}
		}
	}()
}

func listenWebSocket() {
	const address = ":8888"
	const path = "/ws"

	var mux = http.NewServeMux()
	var accept = epoll.NewWsAcceptor(mux, path)
	var app = road.NewApp(accept,
		road.WithSessionRateLimitBySecond(123456789))

	var room = &Room{}
	_ = app.Register(room, component.WithName("room"), component.WithNameFunc(strings.ToLower))
	//testHook(app)

	go func() {
		var pClient = client.New()
		if err := pClient.ConnectToWS(address, path); err != nil {
			logger.Error(err.Error())
		}

		go func() {
			time.Sleep(1 * time.Second)

			for i := 1000; i < 2000; i++ {
				var item = Enter{Name: "kitty", ID: i, Text: text}
				var data = convert.ToJson(item)
				_, err := pClient.SendRequest("room.enter", data)
				if err != nil {
					logger.Error(err.Error())
				}
			}

			for {
				select {
				case msg := <-pClient.MsgChannel():
					if msg != nil {
						if msg.Err {
							logger.Warn(string(msg.Data))
						} else {
							var item Enter
							convert.FromJson(msg.Data, &item)
							logger.Info("id=%d, name=%s, data=%s", item.ID, item.Name, string(msg.Data))
						}
					}
					break
				}
			}
		}()
	}()

	var err = http.ListenAndServe(address, mux)
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
