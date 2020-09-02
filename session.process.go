package road

import (
	"github.com/lixianmin/got/loom"
	"github.com/lixianmin/road/component"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/serialize"
	"github.com/lixianmin/road/service"
	"github.com/lixianmin/road/util"
	"reflect"
)

/********************************************************************
created:    2020-09-01
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func (my *Session) goProcess(later *loom.Later) {
	for {
		select {
		case data := <-my.receivedChan:
			my.processReceived(data)
		case <-my.wc.C():
			return
		}
	}
}

func (my *Session) processReceived(item receivedItem) {
	payload, err := processReceivedImpl(item, my.serializer)
	if item.msg.Type != message.Notify {
		var msg = message.Message{Type: message.Response, ID: item.msg.ID, Data: payload}
		_ = my.sendMessageMayError(item.ctx, msg, err)
	}
}

func processReceivedImpl(data receivedItem, serializer serialize.Serializer) ([]byte, error) {
	handler, err := service.GetHandler(data.route)
	if err != nil {
		return nil, err
	}

	// First unmarshal the handler arg that will be passed to
	// both handler and pipeline functions
	arg, err := unmarshalHandlerArg(handler, serializer, data.msg.Data)
	if err != nil {
		return nil, err
	}

	var args []reflect.Value
	if arg != nil {
		args = []reflect.Value{handler.Receiver, reflect.ValueOf(data.ctx), reflect.ValueOf(arg)}
	} else {
		args = []reflect.Value{handler.Receiver, reflect.ValueOf(data.ctx)}
	}

	resp, err := util.Pcall(handler.Method, args)
	if err != nil {
		return nil, err
	}

	ret, err := util.SerializeOrRaw(serializer, resp)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func unmarshalHandlerArg(handler *component.Handler, serializer serialize.Serializer, payload []byte) (interface{}, error) {
	if handler.IsRawArg {
		return payload, nil
	}

	var arg interface{}
	if handler.Type != nil {
		arg = reflect.New(handler.Type.Elem()).Interface()
		err := serializer.Unmarshal(payload, arg)
		if err != nil {
			return nil, err
		}
	}

	return arg, nil
}
