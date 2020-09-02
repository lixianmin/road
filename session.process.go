package bugfly

import (
	"github.com/lixianmin/bugfly/component"
	"github.com/lixianmin/bugfly/conn/message"
	"github.com/lixianmin/bugfly/serialize"
	"github.com/lixianmin/bugfly/service"
	"github.com/lixianmin/bugfly/util"
	"github.com/lixianmin/got/loom"
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
		_ = my.sendMayError(item.ctx, msg, err)
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

	args := []reflect.Value{handler.Receiver, reflect.ValueOf(data.ctx)}
	if arg != nil {
		args = append(args, reflect.ValueOf(arg))
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
