package chatroom

import (
	"github.com/SevenPlusPlus/gomesh/pkg/network"
	"net"
	"bufio"
	"os"
	"strings"
	"time"
	"fmt"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
)

type ChatClient struct {
	*network.ClientConn
	serverAddr string
	nickName string
}

func NewChatClient(addr, nick string) (* ChatClient){
	network.MsgHandlerRegistry.Register(Chat, ChatClientMsgHandler{})

	rawConn, err := net.Dial("tcp", addr)
	if err != nil {
		log.DefaultLogger.Errorf("Connect to server %s failed %v\n", addr, err)
	}

	onConnectBuilder := network.OnConnect(func(conn network.WriterCloser) bool{
		log.DefaultLogger.Infof("on connect\n")
		return true
	})
	onCloseBuilder := network.OnClose(func(conn network.WriterCloser) {
		log.DefaultLogger.Infof("on close\n")
	})

	conn := network.NewClientConn(rawConn, onConnectBuilder, onCloseBuilder)

	chatClient := ChatClient{
		ClientConn: conn,
		serverAddr: addr,
		nickName: nick,
	}
	return &chatClient
}

func (client *ChatClient)StartClient() {
	defer client.ClientConn.Close()

	client.ClientConn.Start()
	for{
		reader := bufio.NewReader(os.Stdin)
		content, _ := reader.ReadString('\n')
		content = strings.Trim(content, "\n")
		content = strings.Trim(content, "\r")
		if content == "bye" {
			break
		} else {
			msg := ChatMsg{
				NickName: client.nickName,
				Content:  content,
				Icon:     "none",
				Time:     time.Now().UnixNano(),
			}
			if err := client.Write(msg); err != nil {
				log.DefaultLogger.Errorf("Send chat message failed %v", err)
			}
		}
	}
	fmt.Println("goodbye!")
}

