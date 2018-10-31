package network

import (
	"net"
	"sync"
	"golang.org/x/net/context"
	"time"
	"github.com/SevenPlusPlus/gomesh/pkg/types"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
)

func NewCtxWithMsg(ctx context.Context, msg Message)context.Context{
	return context.WithValue(ctx, types.ContextKeyMessage, msg)
}

func MsgFromContext(ctx context.Context) Message {
	return ctx.Value(types.ContextKeyMessage).(Message)
}

func NewCtxWithServer(ctx context.Context, s *Server) context.Context {
	return context.WithValue(ctx, types.ContextKeyServer, s)
}

func ServerFromContext(ctx context.Context) *Server {
	return ctx.Value(types.ContextKeyServer).(*Server)
}

func NewCtxWithNetId(ctx context.Context, netid int64) context.Context {
	return context.WithValue(ctx, types.ContextKeyNetId, netid)
}

func NetIdFromContext(ctx context.Context) int64 {
	return ctx.Value(types.ContextKeyNetId).(int64)
}


type WriterCloser interface {
	Write(Message) error
	Close()
}

type MessageHandlerWrapper struct {
	msg Message
	handler Handler
}

type ServerConn struct {
	netId int64
	server *Server
	rawConn net.Conn
	once *sync.Once
	wg *sync.WaitGroup
	sendDataCh chan []byte
	handlerCh chan MessageHandlerWrapper

	mux sync.Mutex //guard following vars
	name string
	heartbeat int64
	ctx context.Context
	cancel context.CancelFunc
}

func NewServerConn(netId int64, s *Server, conn net.Conn) *ServerConn {
	sc := &ServerConn{
		netId: netId,
		server: s,
		rawConn: conn,
		once: &sync.Once{},
		wg: &sync.WaitGroup{},
		heartbeat: time.Now().UnixNano(),
		sendDataCh: make(chan []byte, s.opts.bufferSize),
		handlerCh: make(chan MessageHandlerWrapper, s.opts.bufferSize),
	}
	sc.ctx, sc.cancel = context.WithCancel(NewCtxWithServer(s.ctx, s))
	sc.name = conn.RemoteAddr().String()
	return sc
}

func (sc *ServerConn) NetId() int64 {
	return sc.netId
}

// SetName sets name of server connection.
func (sc *ServerConn) SetName(name string) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	sc.name = name
}

// Name returns the name of server connection.
func (sc *ServerConn) Name() string {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	name := sc.name
	return name
}

// SetHeartBeat sets the heart beats of server connection.
func (sc *ServerConn) SetHeartBeat(heart int64) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	sc.heartbeat = heart
}

// HeartBeat returns the heart beats of server connection.
func (sc *ServerConn) HeartBeat() int64 {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	heart := sc.heartbeat
	return heart
}

// SetContextValue sets extra data to server connection.
func (sc *ServerConn) SetContextValue(k, v interface{}) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	sc.ctx = context.WithValue(sc.ctx, k, v)
}

// ContextValue gets extra data from server connection.
func (sc *ServerConn) ContextValue(k interface{}) interface{} {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	return sc.ctx.Value(k)
}

func (sc *ServerConn) Start() {
	log.DefaultLogger.Infof("ServerConn starting, %v -> %v\n",
		sc.RemoteAddr(), sc.LocalAddr())
	onConnectCb := sc.server.opts.onConnect
	if onConnectCb != nil {
		onConnectCb(sc)
	}
	loopers := []func(){sc.readLoop, sc.writeLoop, sc.handleLoop}
	for _, looper := range loopers {
		looperFunc := looper
		sc.wg.Add(1)
		go looperFunc()
	}
}

func (sc *ServerConn) Close() {
	sc.once.Do(func() {
		log.DefaultLogger.Infof("ServerConn close gracefully, %v -> %v\n",
			sc.LocalAddr(), sc.RemoteAddr())
		onCloseCb := sc.server.opts.onClose
		if onCloseCb != nil {
			onCloseCb(sc)
		}

		sc.server.connPool.Delete(sc.netId)

		if tcpConn, ok := sc.rawConn.(*net.TCPConn); ok {
			//avoid time-wait state
			tcpConn.SetLinger(0)
		}
		sc.rawConn.Close()

		sc.cancel()

		sc.wg.Wait()
		close(sc.sendDataCh)
		close(sc.handlerCh)

		sc.server.wg.Done()
	})
}

func (sc *ServerConn) Write(msg Message) (errp error) {
	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("ServerConn write msg(%d) failed.\n", msg.MessageType())
			errp = types.ErrServerConnClosed
		}
	}()
	var packet []byte
	packet, err := sc.server.opts.codec.Encode(msg)
	if err != nil {
		log.DefaultLogger.Errorf("ServerConn encode msg(%d) failed %v\n",
			msg.MessageType(), err)
		return err
	}

	select {
	case sc.sendDataCh <- packet:
		errp = nil
	default:
		errp = types.ErrWouldBlock
	}
	return
}

// LocalAddr returns the local address of server connection.
func (sc *ServerConn) LocalAddr() net.Addr {
	return sc.rawConn.LocalAddr()
}

// RemoteAddr returns the remote address of server connection.
func (sc *ServerConn) RemoteAddr() net.Addr {
	return sc.rawConn.RemoteAddr()
}

func (sc *ServerConn) readLoop() {
	rawConn := sc.rawConn
	codec := sc.server.opts.codec
	connDone := sc.ctx.Done()
	serverDone := sc.server.ctx.Done()
	onMessageCb := sc.server.opts.onMessage
	var msg Message
	var err error

	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("Panics: %v\n", p)
		}
		sc.wg.Done()
		log.DefaultLogger.Debugf("ServerConn readLoop go-routine exited\n")
		sc.Close()
	}()

	for{
		select {
		case <-connDone:
			log.DefaultLogger.Infof("receiving cancel signal from server-conn")
			return
		case <-serverDone:
			log.DefaultLogger.Infof("receiving cancel signal from server")
			return
		default:
			msg, err = codec.Decode(rawConn)
			if err != nil {
				log.DefaultLogger.Errorf("Error occurred while ServerConn decoding message %v\n", err)
				return
			}
			sc.SetHeartBeat(time.Now().UnixNano())
			handler := MsgHandlerRegistry.GetMsgHandler(msg.MessageType())
			if handler == nil {
				if onMessageCb != nil {
					log.DefaultLogger.Infof("msg %d call onMessage()\n", msg.MessageType())
					onMessageCb(msg, sc)
				} else {
					log.DefaultLogger.Infof("no handler or onMessage found for message %d\n", msg.MessageType())
				}
				continue
			}
			sc.handlerCh <- MessageHandlerWrapper{msg, handler}
		}
	}
}

func (sc *ServerConn) writeLoop() {
	rawConn := sc.rawConn
	connDone := sc.ctx.Done()
	serverDone := sc.server.ctx.Done()
	var packet []byte
	var err error

	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("panics: %v\n", p)
		}
	OuterFor:
		for{
			select {
			case packet = <- sc.sendDataCh:
				if packet != nil {
					if _, err = rawConn.Write(packet); err != nil{
						log.DefaultLogger.Errorf("Error occurred while sending data %v\n", err)
					}
				}
			default:
				break OuterFor
			}
		}
		sc.wg.Done()
		log.DefaultLogger.Infof("ServerConn writeLoop go-routine exited\n")
		sc.Close()
	}()

	for{
		select{
		case <-connDone:
			log.DefaultLogger.Infof("receiving cancel signal from server-conn")
			return
		case <-serverDone:
			log.DefaultLogger.Infof("receiving cancel signal from server")
			return
		case packet = <-sc.sendDataCh:
			if packet != nil {
				if _, err = rawConn.Write(packet); err != nil{
					log.DefaultLogger.Errorf("Error occurred while sending data %v\n", err)
					return
				}
			}
		}
	}
}

func(sc *ServerConn)handleLoop(){
	connDone := sc.ctx.Done()
	serverDone := sc.server.ctx.Done()

	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("Panics: %v\n", p)
		}
		sc.wg.Done()
		log.DefaultLogger.Infof("ServerConn handleLoop go-routine exited\n")
		sc.Close()
	}()

	var msgHandler MessageHandlerWrapper
	for{
		select{
		case <-connDone:
			log.DefaultLogger.Infof("receiving cancel signal from server-conn")
			return
		case <-serverDone:
			log.DefaultLogger.Infof("receiving cancel signal from server")
			return
		case msgHandler = <-sc.handlerCh:
			msg, handler := msgHandler.msg, msgHandler.handler
			if handler != nil {
				workerCb := func() {
					handler.Handle(NewCtxWithNetId(NewCtxWithMsg(sc.ctx, msg), sc.netId), sc)
				}
				if err := sc.server.workerPool.Put(sc.netId, workerCb); err != nil {
					log.DefaultLogger.Errorf("ServerConn put handler to workerPool failed %v\n", err)
				}
			}
		}
	}
}