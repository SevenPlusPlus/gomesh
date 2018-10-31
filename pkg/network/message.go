package network

import (
	"bytes"
	"encoding/binary"
	"context"
)

type MsgType int32

const (
	Heartbeat MsgType = iota
)

type MessageCodec interface {
	Serialize(obj interface{})([]byte, error)
	Deserialize(data []byte)(interface{}, error)
}

type Message interface {
	MessageType() MsgType
}

type HeartbeatMessage struct {
	Timestamp int64
}

func(HeartbeatMessage) MessageType() MsgType{
	return Heartbeat
}

type HeartbeatMessageCodec struct{}

func(codec *HeartbeatMessageCodec)Serialize(obj interface{})([]byte, error) {
	buf := new(bytes.Buffer)
	msg, _ := obj.(HeartbeatMessage)
	if err := binary.Write(buf, binary.LittleEndian, msg.Timestamp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func(codec *HeartbeatMessageCodec)Deserialize(data []byte)(interface{}, error) {
	buf := bytes.NewReader(data)
	var myts int64
	if err := binary.Read(buf, binary.LittleEndian, &myts); err != nil {
		return nil, err
	}
	return HeartbeatMessage{
		Timestamp: myts,
	}, nil
}

type HeartbeatMessageHandler struct{}
func(hanlder *HeartbeatMessageHandler)Handle(ctx context.Context, c WriterCloser) {
	msg := MsgFromContext(ctx)
	switch c := c.(type){
	case *ServerConn:
		c.SetHeartBeat(msg.(HeartbeatMessage).Timestamp)
	case *ClientConn:
		c.SetHeartBeat(msg.(HeartbeatMessage).Timestamp)
	}
}

