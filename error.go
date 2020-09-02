package road

import "fmt"

/********************************************************************
created:    2020-09-02
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (err *Error) Error() string {
	return fmt.Sprintf("code=%q message=%q", err.Code, err.Message)
}

func checkCreateError(err error) *Error {
	if err1, ok := err.(*Error); ok {
		return err1
	}

	var err2 = &Error{
		Code:    "unknown",
		Message: err.Error(),
	}

	return err2
}
