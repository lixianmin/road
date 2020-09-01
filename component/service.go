package component

import (
	"errors"
	"github.com/lixianmin/gonsole/bugfly/conn/message"
	"reflect"
)

/********************************************************************
created:    2020-08-29
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type (
	//Handler represents a message.Message's handler's meta information.
	Handler struct {
		Receiver    reflect.Value  // receiver of method
		Method      reflect.Method // method stub
		Type        reflect.Type   // low-level type of method
		IsRawArg    bool           // whether the data need to serialize
		MessageType message.Type   // handler allowed message type (either request or notify)
	}

	// Service implements a specific service, some of it's methods will be
	// called when the correspond events is occurred.
	Service struct {
		Name     string              // name of service
		Type     reflect.Type        // type of the receiver
		Receiver reflect.Value       // receiver of methods for the service
		Handlers map[string]*Handler // registered methods
		Options  options             // options
	}
)

// NewService creates a new service
func NewService(comp Component, opts []Option) *Service {
	s := &Service{
		Type:     reflect.TypeOf(comp),
		Receiver: reflect.ValueOf(comp),
	}

	// apply options
	for i := range opts {
		opt := opts[i]
		opt(&s.Options)
	}
	if name := s.Options.name; name != "" {
		s.Name = name
	} else {
		s.Name = reflect.Indirect(s.Receiver).Type().Name()
	}

	return s
}

// ExtractHandler extract the set of methods from the
// receiver value which satisfy the following conditions:
// - exported method of exported type
// - one or two arguments
// - the first argument is context.Context
// - the second argument (if it exists) is []byte or a pointer
// - zero or two outputs
// - the first output is [] or a pointer
// - the second output is an error
func (s *Service) ExtractHandler() error {
	typeName := reflect.Indirect(s.Receiver).Type().Name()
	if typeName == "" {
		return errors.New("no service name for type " + s.Type.String())
	}

	if !isExported(typeName) {
		return errors.New("type " + typeName + " is not exported")
	}

	// Install the methods
	s.Handlers = suitableHandlerMethods(s.Type, s.Options.nameFunc)

	if len(s.Handlers) == 0 {
		str := ""
		// To help the user, see if a pointer receiver would work.
		method := suitableHandlerMethods(reflect.PtrTo(s.Type), s.Options.nameFunc)
		if len(method) != 0 {
			str = "type " + s.Name + " has no exported methods of handler type (hint: pass a pointer to value of that type)"
		} else {
			str = "type " + s.Name + " has no exported methods of handler type"
		}
		return errors.New(str)
	}

	for i := range s.Handlers {
		s.Handlers[i].Receiver = s.Receiver
	}

	return nil
}
