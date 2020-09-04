package road

import (
	"encoding/json"
	"github.com/lixianmin/road/conn/codec"
	"github.com/lixianmin/road/conn/message"
	"github.com/lixianmin/road/conn/packet"
	"github.com/lixianmin/road/serialize"
	"github.com/lixianmin/road/util/compression"
	"time"
)

/********************************************************************
created:    2020-09-04
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type commonSessionArgs struct {
	packetEncoder  codec.PacketEncoder
	packetDecoder  codec.PacketDecoder
	messageEncoder message.Encoder
	serializer     serialize.Serializer

	heartbeatTimeout      time.Duration
	heartbeatPacketData   []byte
	handshakeResponseData []byte
}

func newCommonSessionArgs(dataCompression bool, heartbeatTimeout time.Duration) *commonSessionArgs {
	var my = &commonSessionArgs{
		packetDecoder:    codec.NewPomeloPacketDecoder(),
		packetEncoder:    codec.NewPomeloPacketEncoder(),
		messageEncoder:   message.NewMessagesEncoder(dataCompression),
		serializer:       serialize.NewJsonSerializer(),
		heartbeatTimeout: heartbeatTimeout,
	}

	my.heartbeatPacketData = my.encodeHeartbeatData()
	my.handshakeResponseData = my.encodeHandshakeData(dataCompression)

	return my
}

func (my *commonSessionArgs) encodeHeartbeatData() []byte {
	var bytes, err = my.packetEncoder.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}

	return bytes
}

func (my *commonSessionArgs) encodeHandshakeData(dataCompression bool) []byte {
	hData := map[string]interface{}{
		"code": 200,
		"sys": map[string]interface{}{
			"heartbeat":  my.heartbeatTimeout.Seconds(),
			"dict":       message.GetDictionary(),
			"serializer": my.serializer.GetName(),
		},
	}

	data, err := json.Marshal(hData)
	if err != nil {
		panic(err)
	}

	if dataCompression {
		compressedData, err := compression.DeflateData(data)
		if err != nil {
			panic(err)
		}

		if len(compressedData) < len(data) {
			data = compressedData
		}
	}

	bytes, err := my.packetEncoder.Encode(packet.Handshake, data)
	if err != nil {
		panic(err)
	}

	return bytes
}
