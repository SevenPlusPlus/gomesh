package network

import (
	"golang.org/x/net/context"
	"sync"
	"fmt"
)

type HandlerRegistry struct {
	handlerMap map[MsgType]MessageHandler
	mux sync.Mutex
}

var MsgHandlerRegistry HandlerRegistry

type Handler interface {
	Handle(context.Context, WriterCloser)
}

type MessageHandler interface {
	Handler
	MessageCodec
}

func init() {
	MsgHandlerRegistry = HandlerRegistry{
		handlerMap: make(map[MsgType]MessageHandler),
	}
}

func(registry *HandlerRegistry)Register(msgtype MsgType, handler MessageHandler) {
	registry.mux.Lock()
	defer registry.mux.Unlock()
	if _, ok := registry.handlerMap[msgtype]; ok {
		panic(fmt.Sprintf("trying to register message %d twice", msgtype))
	}
	registry.handlerMap[msgtype] = handler
}

func(registry *HandlerRegistry)GetMsgCodec(msgType MsgType)MessageCodec {
	registry.mux.Lock()
	defer registry.mux.Unlock()
	item, ok := registry.handlerMap[msgType]
	if !ok {
		return nil
	}
	return item
}

func(registry *HandlerRegistry)GetMsgHandler(msgType MsgType)Handler {
	registry.mux.Lock()
	defer registry.mux.Unlock()
	item, ok := registry.handlerMap[msgType]
	if !ok {
		return nil
	}
	return item
}