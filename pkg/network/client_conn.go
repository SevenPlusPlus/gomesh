package network

import (
	"net"
	"sync"
	"golang.org/x/net/context"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
	"time"
	"github.com/SevenPlusPlus/gomesh/pkg/types"
	"crypto/tls"
)

type ClientConn struct {
	raddr string
	opts options
	rawConn net.Conn
	once *sync.Once
	wg *sync.WaitGroup
	sendDataCh chan []byte
	handlerCh chan MessageHandlerWrapper
	mux sync.Mutex //protect following vars
	name string
	heartbeat int64
	ctx context.Context
	cancel context.CancelFunc
}

func NewClientConn(conn net.Conn, builders ...OptionBuilder) *ClientConn {
	var opts options
	for _, builder := range builders {
		builder(&opts)
	}
	if opts.codec == nil {
		opts.codec = TypeLengthValueCodec{}
	}
	if opts.bufferSize <= 0 {
		opts.bufferSize = BufferSize256
	}
	return newClientConnWithOptions(conn, opts)
}

func newClientConnWithOptions(conn net.Conn, opts options) *ClientConn {
	cc := &ClientConn{
		raddr: conn.RemoteAddr().String(),
		opts: opts,
		rawConn: conn,
		once: &sync.Once{},
		wg: &sync.WaitGroup{},
		sendDataCh: make(chan []byte),
		handlerCh: make(chan MessageHandlerWrapper),
		heartbeat: time.Now().UnixNano(),
	}
	cc.ctx,cc.cancel = context.WithCancel(context.Background())
	cc.name = conn.RemoteAddr().String()
	return cc
}

// SetName sets the name of client connection.
func (cc *ClientConn) SetName(name string) {
	cc.mux.Lock()
	cc.name = name
	cc.mux.Unlock()
}

// Name gets the name of client connection.
func (cc *ClientConn) Name() string {
	cc.mux.Lock()
	name := cc.name
	cc.mux.Unlock()
	return name
}

// SetHeartBeat sets the heart beats of client connection.
func (cc *ClientConn) SetHeartBeat(heart int64) {
	cc.mux.Lock()
	cc.heartbeat = heart
	cc.mux.Unlock()
}

// HeartBeat gets the heart beats of client connection.
func (cc *ClientConn) HeartBeat() int64 {
	cc.mux.Lock()
	heart := cc.heartbeat
	cc.mux.Unlock()
	return heart
}

// SetContextValue sets extra data to client connection.
func (cc *ClientConn) SetContextValue(k, v interface{}) {
	cc.mux.Lock()
	defer cc.mux.Unlock()
	cc.ctx = context.WithValue(cc.ctx, k, v)
}

// ContextValue gets extra data from client connection.
func (cc *ClientConn) ContextValue(k interface{}) interface{} {
	cc.mux.Lock()
	defer cc.mux.Unlock()
	return cc.ctx.Value(k)
}

func (cc *ClientConn) Write(msg Message) (errp error) {
	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("ClientConn write msg-(%d) failed\n", msg.MessageType())
			errp = types.ErrServerConnClosed
		}
	}()
	var packet []byte
	packet, err := cc.opts.codec.Encode(msg)
	if err != nil {
		log.DefaultLogger.Errorf("ClientConn encode msg-(%d) failed\n", msg.MessageType())
		return err
	}

	select {
	case cc.sendDataCh <- packet:
		errp = nil
	default:
		errp = types.ErrWouldBlock
	}
	return
}

func (cc *ClientConn) Start() {
	log.DefaultLogger.Infof("Client conn starting, %v -> %v\n", cc.LocalAddr(), cc.RemoteAddr())
	onConnectCb := cc.opts.onConnect
	if onConnectCb != nil {
		onConnectCb(cc)
	}

	loopers := []func(){cc.readLoop, cc.writeLoop, cc.handleLoop}
	for _, loop := range loopers {
		cc.wg.Add(1)
		loopFunc := loop
		go loopFunc()
	}
}

func (cc *ClientConn) Close() {
	cc.once.Do(func() {
		log.DefaultLogger.Infof("Client conn close gracefully %v -> %v\n",
			cc.LocalAddr(), cc.RemoteAddr())
		onCloseCb := cc.opts.onClose
		if onCloseCb != nil {
			onCloseCb(cc)
		}

		cc.rawConn.Close()
		cc.cancel()

		cc.wg.Wait()
		close(cc.sendDataCh)
		close(cc.handlerCh)

		if cc.opts.reconnect {
			cc.reconnect()
		}
	})
}

func (cc *ClientConn) RemoteAddr() net.Addr {
	return cc.rawConn.RemoteAddr()
}

func (cc *ClientConn) LocalAddr() net.Addr {
	return cc.rawConn.LocalAddr()
}

func (cc *ClientConn) reconnect() {
	var c net.Conn
	var err error
	if cc.opts.tlsConfig != nil {
		c, err = tls.Dial("tcp", cc.raddr, cc.opts.tlsConfig)
		if err != nil {
			log.DefaultLogger.Errorf("tls dail server %s failed %v\n", cc.raddr, err)
		}
	} else {
		c, err = net.Dial("tcp", cc.raddr)
		if err != nil {
			log.DefaultLogger.Errorf("net dial %s failed %v\n", cc.raddr, err)
		}
	}
	*cc = *newClientConnWithOptions(c, cc.opts)
	cc.Start()
}

func (cc *ClientConn) readLoop() {
	rawConn := cc.rawConn
	codec := cc.opts.codec
	connDone := cc.ctx.Done()
	onMessageCb := cc.opts.onMessage
	var msg Message
	var err error

	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("Panics: %v\n", p)
		}
		cc.wg.Done()
		log.DefaultLogger.Debugf("readLoop go-routine exited\n")
		cc.Close()
	}()

	for{
		select {
		case <-connDone:
			log.DefaultLogger.Infof("receiving cancel signal from client-conn")
			return
		default:
			msg, err = codec.Decode(rawConn)
			if err != nil {
				log.DefaultLogger.Errorf("Error occured while decoding message %v\n", err)
				return
			}
			cc.SetHeartBeat(time.Now().UnixNano())
			handler := msgHandlerRegistry.GetMsgHandler(msg.MessageType())
			if handler == nil {
				if onMessageCb != nil {
					log.DefaultLogger.Infof("msg %d call onMessage()\n", msg.MessageType())
					onMessageCb(msg, cc)
				} else {
					log.DefaultLogger.Infof("no handler or onMessage found for message %d\n", msg.MessageType())
				}
				continue
			}
			cc.handlerCh <- MessageHandlerWrapper{msg, handler}
		}
	}
}

func (cc *ClientConn)writeLoop() {
	rawConn := cc.rawConn
	connDone := cc.ctx.Done()
	var packet []byte
	var err error

	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("panics: %v\n", p)
		}
	OuterFor:
		for{
			select {
			case packet = <- cc.sendDataCh:
				if packet != nil {
					if _, err = rawConn.Write(packet); err != nil{
						log.DefaultLogger.Errorf("Error occured while sending data %v\n", err)
					}
				}
			default:
				break OuterFor
			}
		}
		cc.wg.Done()
		log.DefaultLogger.Infof("writeLoop go-routine exited\n")
		cc.Close()
	}()

	for{
		select{
		case <-connDone:
			log.DefaultLogger.Infof("receiving cancel signal from client-conn")
			return
		case packet = <-cc.sendDataCh:
			if packet != nil {
				if _, err = rawConn.Write(packet); err != nil{
					log.DefaultLogger.Errorf("Error occured while sending data %v\n", err)
					return
				}
			}
		}
	}
}

func(cc *ClientConn) handleLoop() {
	connDone := cc.ctx.Done()

	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("Panics: %v\n", p)
		}
		cc.wg.Done()
		log.DefaultLogger.Infof("handleLoop go-routine exited\n")
		cc.Close()
	}()

	var msgHandler MessageHandlerWrapper
	for {
		select {
		case <-connDone:
			log.DefaultLogger.Infof("receiving cancel signal from client-conn")
			return
		case msgHandler = <-cc.handlerCh:
			msg, handler := msgHandler.msg, msgHandler.handler
			if handler != nil {
				handler.Handle(NewCtxWithMsg(cc.ctx, msg), cc)
			}
		}
	}
}