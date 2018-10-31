package chatroom

import (
	"github.com/SevenPlusPlus/gomesh/pkg/network"
	"encoding/json"
	"golang.org/x/net/context"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
)

const (
	Chat network.MsgType = 1
)

type ChatMsg struct {
	NickName string `json:"nick"`   //对应JSON的nick
	Content string	`json:"body,omitempty"`  //如果为空则忽略该字段
	Icon string `json:"-"`   //忽略该字段
	Time int64	`json:"time"` //对应JSON的time
}

func (chat ChatMsg) MessageType() network.MsgType{
	return Chat
}

type ChatMsgCodec struct{}

func(codec ChatMsgCodec)Serialize(obj interface{})([]byte, error) {
	chatMsg := obj.(ChatMsg)
	return json.Marshal(chatMsg)
}

func(codec ChatMsgCodec)Deserialize(data []byte)(interface{}, error) {
	var chatMsg ChatMsg
	err := json.Unmarshal(data, &chatMsg)
	return chatMsg, err
}

type ChatRoomMsgHandler struct{
	 ChatMsgCodec
}

func (ChatRoomMsgHandler) Handle(ctx context.Context, conn network.WriterCloser) {
	log.DefaultLogger.Infof("Process room chat message.\n")
	server := network.ServerFromContext(ctx)
	if server != nil {
		msg := network.MsgFromContext(ctx)
		server.Broadcast(msg)
	}
}

type ChatClientMsgHandler struct {
	ChatMsgCodec
}

func (ChatClientMsgHandler) Handle(ctx context.Context, conn network.WriterCloser) {
	log.DefaultLogger.Infof("Process client chat message.\n")
	msg := network.MsgFromContext(ctx)
	chatMsg := msg.(ChatMsg)
	log.DefaultLogger.Infof("Got message %s from %s\n", chatMsg.Content, chatMsg.NickName)
}