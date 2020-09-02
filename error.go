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

func NewError(code string, format string, args ...interface{}) *Error {
	var err = &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}

	return err
}

func (err *Error) Error() string {
	return fmt.Sprintf("code=%q message=%q", err.Code, err.Message)
}

func checkCreateError(err error) *Error {
	if err1, ok := err.(*Error); ok {
		return err1
	}

	var err2 = &Error{
		Code:    "PlainError",
		Message: err.Error(),
	}

	return err2
}
