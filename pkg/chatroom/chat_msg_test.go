package chatroom

import (
	"testing"
	"time"
	"fmt"
)

func TestChatMsgCodec(t *testing.T) {
	msgCodec := ChatMsgCodec{}
	ChatMsg := ChatMsg{
		NickName: "maven",
		Content: "hello",
		Time: time.Now().UnixNano(),
	}
	bytes,_ := msgCodec.Serialize(ChatMsg)
	str := string(bytes)
	fmt.Println(str)

	newChat, _ := msgCodec.Deserialize(bytes)
	fmt.Println(newChat)
}
