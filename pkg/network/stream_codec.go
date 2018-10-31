package network

import (
	"net"
	"bytes"
	"encoding/binary"
	"io"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
)

const (
	MessageTypeBytes = 4
	MessageLenBytes = 4
	MessageMaxBytes = 1 << 23
)

type StreamCodec interface {
	Encode(Message)([]byte, error)
	Decode(net.Conn)(Message, error)
}

// Format: type-length-value |4 bytes|4 bytes|n bytes <= 8M|
type TypeLengthValueCodec struct{}

func(codec TypeLengthValueCodec)Decode(raw net.Conn)(Message, error) {
	typeData := make([]byte, MessageTypeBytes)
	_, err := io.ReadFull(raw, typeData)
	if err != nil {
		log.DefaultLogger.Errorf("read message type failed:%v\n", err)
		return nil, err
	}
	typeBuf := bytes.NewReader(typeData)
	var msgType int32
	if err := binary.Read(typeBuf, binary.LittleEndian, &msgType); err != nil {
		return nil, err
	}
	msgCodec:= MsgHandlerRegistry.GetMsgCodec(MsgType(msgType))
	lengthBytes := make([]byte, MessageLenBytes)
	if _, err := io.ReadFull(raw, lengthBytes); err != nil {
		return nil, err
	}
	lengthBuf := bytes.NewReader(lengthBytes)
	var msgLen int32
	if err := binary.Read(lengthBuf, binary.LittleEndian, &msgLen); err != nil {
		return nil, err
	}
	if msgLen > MessageMaxBytes {
		log.DefaultLogger.Errorf("message has bytes(%d) over max len %d\n",
			msgLen, MessageMaxBytes)
	}
	// read application data
	msgBytes := make([]byte, msgLen)
	_, err = io.ReadFull(raw, msgBytes)
	if err != nil {
		return nil, err
	}
	msg, err := msgCodec.Deserialize(msgBytes)
	return msg.(Message), nil
}

func(codec TypeLengthValueCodec)Encode(msg Message)([]byte, error) {
	msgCodec := MsgHandlerRegistry.GetMsgCodec(msg.MessageType())
	data, err := msgCodec.Serialize(msg)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, msg.MessageType())
	binary.Write(buf, binary.LittleEndian, int32(len(data)))
	buf.Write(data)
	return buf.Bytes(), nil
}
