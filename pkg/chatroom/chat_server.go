package chatroom

import (
	"github.com/SevenPlusPlus/gomesh/pkg/network"
	"net"
	"os"
	"os/signal"
	"syscall"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
)

type ChatServer struct{
	*network.Server
}

func NewChatRoomServer() *ChatServer {
	onConnectBuilder := network.OnConnect(func(conn network.WriterCloser) bool{
		log.DefaultLogger.Infof("on connect\n")
		return true
	})
	onCloseBuilder := network.OnClose(func(conn network.WriterCloser) {
		log.DefaultLogger.Infof("on close\n")
	})
	return &ChatServer{
		Server: network.NewServer(onConnectBuilder, onCloseBuilder),
	}
}

func (server *ChatServer) StartServer(bindAddr string){
	network.MsgHandlerRegistry.Register(Chat, ChatRoomMsgHandler{})

	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		log.DefaultLogger.Errorf("Listen error %v\n", err)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		server.Stop()
	}()

	server.Start(listener)
}