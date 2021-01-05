package util

import (
	"github.com/lixianmin/logo"
	"github.com/lixianmin/road/ifs"
	"github.com/lixianmin/road/serialize"
	"reflect"
)

/********************************************************************
created:    2020-08-29
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

// PCall calls a method that returns an interface and an error and recovers in case of panic
func PCall(method reflect.Method, args []reflect.Value) (rets interface{}, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			logo.Error("methodName=%d, recover=%v", method.Name, rec)
		}
	}()

	r := method.Func.Call(args)

	// r can have 0 length in case of notify handlers
	// otherwise it will have 2 outputs: an interface and an error
	if len(r) == 2 {
		if v := r[1].Interface(); v != nil {
			err = v.(error)
		} else if !r[0].IsNil() {
			rets = r[0].Interface()
		} else {
			err = ifs.ErrReplyShouldBeNotNull
		}
	}
	return
}

// SerializeOrRaw serializes the interface if its not an array of bytes already
func SerializeOrRaw(serializer serialize.Serializer, v interface{}) ([]byte, error) {
	if data, ok := v.([]byte); ok {
		return data, nil
	}

	data, err := serializer.Marshal(v)
	if err != nil {
		return nil, err
	}

	return data, nil
}
