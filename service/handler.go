package service

import (
	"fmt"
	"github.com/lixianmin/road/docgenerator"
	"github.com/lixianmin/road/logger"
	"github.com/lixianmin/road/component"
)

/********************************************************************
created:    2020-08-29
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

var (
	handlers = make(map[string]*component.Handler) // all handler method
)

type HandlerService struct {
	services map[string]*component.Service // all registered service
}

func NewHandlerService() *HandlerService {
	var my = &HandlerService{
		services: make(map[string]*component.Service),
	}

	return my
}

func (my *HandlerService) Register(comp component.Component, opts []component.Option) error {
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
		var route = fmt.Sprintf("%s.%s", s.Name, name)
		handlers[route] = handler
		logger.Debug("route=%s", route)
	}

	return nil
}

// Docs returns documentation for handlers
func (my *HandlerService) Docs(getPtrNames bool) (map[string]interface{}, error) {
	if my == nil {
		return map[string]interface{}{}, nil
	}

	return docgenerator.HandlersDocs("game", my.services, getPtrNames)
}
