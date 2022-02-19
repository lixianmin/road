package road

import (
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/util"
)

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func (my *Session) Push(route string, v interface{}) error {
	if my.wc.IsClosed() {
		return nil
	}

	var payload, err = util.SerializeOrRaw(my.app.serializer, v)
	var msg = message.Message{Type: message.Push, Route: route, Data: payload}
	var data, err1 = my.encodeMessageMayError(msg, err)
	if err1 != nil {
		return err1
	}

	//select {
	//case my.sendingChan <- data:
	//case <-my.wc.C():
	//}
	err = my.writeBytes(data)
	return err
}

// 强踢下线
func (my *Session) Kick() error {
	if my.wc.IsClosed() {
		return nil
	}

	p, err := my.app.packetEncoder.Encode(packet.Kick, nil)
	if err != nil {
		return err
	}

	return my.writeBytes(p)
	//select {
	//case my.sendingChan <- p:
	//	logo.Info("session(%d) will be closed by Kick()", my.id)
	//case <-my.wc.C():
	//}
	//return nil
}

func (my *Session) encodeMessageMayError(msg message.Message, err error) ([]byte, error) {
	if err != nil {
		msg.Err = true
		//logo.Info("process failed, route=%s, err=%q", msg.Route, err.Error())

		// err需要支持json序列化的话，就不能是一个简单的字符串
		var errWrap = checkCreateError(err)

		var err1 error
		msg.Data, err1 = util.SerializeOrRaw(my.app.serializer, errWrap)
		if err1 != nil {
			logo.Info("serialize failed, route=%s, err1=%q", msg.Route, err1.Error())
			return nil, err1
		}
	}

	data, err2 := my.packetEncodeMessage(&msg)
	if err2 != nil {
		logo.Info("send failed, route=%s, err2=%q", msg.Route, err2.Error())
		return nil, err2
	}

	return data, nil
}

func (my *Session) writeBytes(data []byte) error {
	var size = len(data)
	//logo.Debug("len(data)=%d, data=%v", size, data)
	if size > 0 {
		var item = sendingItem{session: my, data: data}
		my.sender.sendingChan <- item
	}

	return nil
}

//func (my *Session) writeBytes(data []byte) error {
//	if len(data) > 0 {
//		var _, err = my.conn.Write(data)
//		return err
//	}
//
//	return nil
//}

func (my *Session) packetEncodeMessage(msg *message.Message) ([]byte, error) {
	data, err := my.app.messageEncoder.Encode(msg)
	if err != nil {
		return nil, err
	}

	// packet encode
	p, err := my.app.packetEncoder.Encode(packet.Data, data)
	if err != nil {
		return nil, err
	}

	return p, nil
}
