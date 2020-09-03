package acceptor

import (
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/ifs"
	"io"
	"io/ioutil"
	"net"
)

/********************************************************************
created:    2020-09-03
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type tcpConn struct {
	net.Conn
}

// GetNextMessage reads the next message available in the stream
func (t *tcpConn) GetNextMessage() (b []byte, err error) {
	header, err := ioutil.ReadAll(io.LimitReader(t.Conn, codec.HeadLength))
	if err != nil {
		return nil, err
	}

	msgSize, _, err := codec.ParseHeader(header)
	if err != nil {
		return nil, err
	}

	msgData, err := ioutil.ReadAll(io.LimitReader(t.Conn, int64(msgSize)))
	if err != nil {
		return nil, err
	}

	if len(msgData) < msgSize {
		return nil, ifs.ErrReceivedMsgSmallerThanExpected
	}

	return append(header, msgData...), nil
}
