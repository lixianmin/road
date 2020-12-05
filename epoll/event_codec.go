package epoll

import (
	"github.com/lixianmin/road/ifs"
	"github.com/panjf2000/gnet"
)

/********************************************************************
created:    2020-12-05
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type EventCodec struct {
}

func newEventCodec() *EventCodec {
	var my = &EventCodec{}
	return my
}

func (my *EventCodec) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	return buf, nil
}

func (my *EventCodec) Decode(c gnet.Conn) ([]byte, error) {
	var (
		in     innerBuffer
		header []byte
		err    error
	)

	in = c.Read()

	const lengthFieldOffset = 1
	header, err = in.readN(lengthFieldOffset)
	if err != nil {
		return nil, ifs.ErrUnexpectedEOF
	}

	lenBuf, frameLength, err := my.getUnadjustedFrameLength(&in)
	if err != nil {
		return nil, err
	}

	// real message length
	msgLength := int(frameLength)
	msg, err := in.readN(msgLength)
	if err != nil {
		return nil, ifs.ErrUnexpectedEOF
	}

	var fullMessage = make([]byte, len(header)+len(lenBuf)+msgLength)
	copy(fullMessage, header)
	copy(fullMessage[len(header):], lenBuf)
	copy(fullMessage[len(header)+len(lenBuf):], msg)

	c.ShiftN(len(fullMessage))
	return fullMessage, nil
}

func (my *EventCodec) getUnadjustedFrameLength(in *innerBuffer) ([]byte, uint64, error) {
	const lengthFieldSize = 3
	lenBuf, err := in.readN(lengthFieldSize)
	if err != nil {
		return nil, 0, ifs.ErrUnexpectedEOF
	}

	return lenBuf, readUint24BigEndian(lenBuf), nil
}

func readUint24BigEndian(b []byte) uint64 {
	return uint64(b[2]) | uint64(b[1])<<8 | uint64(b[0])<<16
}
