package main

import "context"

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type Enter struct {
	ID   int	`json:"id"`
	Name string	`json:"name"`
	Text string	`json:"text"`
}

type EnterRe struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Room struct {
}

func (room *Room) Enter(ctx context.Context, request *Enter) (*EnterRe, error) {
	return &EnterRe{
		ID:   request.ID,
		Name: request.Name,
	}, nil
}
