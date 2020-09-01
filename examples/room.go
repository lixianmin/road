package main

import "context"

/********************************************************************
created:    2020-08-31
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type Enter struct {
	ID   int
	Name string
}

type EnterRe struct {
	RID  int
	Name string
}

type Room struct {
}

func (room *Room) Enter(ctx context.Context, request *Enter) (*EnterRe, error) {
	return &EnterRe{
		RID:  request.ID,
		Name: request.Name,
	}, nil
}
