package epoll

import (
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/ifs"
	"net"
	"reflect"
)

/********************************************************************
created:    2020-09-06
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

func socketFD(conn net.Conn) int64 {
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")

	return pfdVal.FieldByName("Sysfd").Int()
}

func checkReceivedMsgBytes(msgBytes []byte) (error) {
	if len(msgBytes) < codec.HeadLength {
		return packet.ErrInvalidPomeloHeader
	}

	header := msgBytes[:codec.HeadLength]
	msgSize, _, err := codec.ParseHeader(header)
	if err != nil {
		return err
	}

	dataLen := len(msgBytes[codec.HeadLength:])
	if dataLen < msgSize {
		return ifs.ErrReceivedMsgSmallerThanExpected
	} else if dataLen > msgSize {
		return ifs.ErrReceivedMsgBiggerThanExpected
	}

	return nil
}
