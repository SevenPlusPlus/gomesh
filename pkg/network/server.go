package network

import (
	"crypto/tls"
	"golang.org/x/net/context"
	"sync"
	"net"
	"github.com/SevenPlusPlus/gomesh/pkg/utils"
	"time"
	"crypto/rand"
	"fmt"
	"github.com/SevenPlusPlus/gomesh/pkg/types"
	"os"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
)

const (
	MaxConnections = 1000
	BufferSize256 = 256
	DefaultWorkerPoolSize = 20
)

type OnConnectCallback func(WriterCloser) bool
type OnMessageCallback func(Message, WriterCloser)
type OnCloseCallback func(WriterCloser)
type OnErrorCallback func(WriterCloser)

type options struct {
	tlsConfig *tls.Config
	codec StreamCodec
	onConnect OnConnectCallback
	onMessage OnMessageCallback
	onClose OnCloseCallback
	onError OnErrorCallback
	workerPoolSize int
	bufferSize int
	reconnect bool
}

type OptionBuilder func(options *options)

func ReconnnectOption() OptionBuilder {
	return func(options *options) {
		options.reconnect = true
	}
}

func CodecOption(codec StreamCodec) OptionBuilder {
	return func(options *options) {
		options.codec = codec
	}
}

func TlsOption(config *tls.Config) OptionBuilder {
	return func(options *options) {
		options.tlsConfig = config
	}
}

func WorkerPoolSize(workerSize int) OptionBuilder {
	return func(options *options) {
		options.workerPoolSize = workerSize
	}
}

func ChannelBufferSize(bufferSize int) OptionBuilder {
	return func(options *options) {
		options.bufferSize = bufferSize
	}
}

func OnConnect(cb OnConnectCallback) OptionBuilder {
	return func(options *options) {
		options.onConnect = cb
	}
}

func OnMessage(cb OnMessageCallback) OptionBuilder {
	return func(options *options) {
		options.onMessage = cb
	}
}

func OnClose(cb OnCloseCallback) OptionBuilder {
	return func(options *options) {
		options.onClose = cb
	}
}

func OnError(cb OnErrorCallback) OptionBuilder {
	return func(options *options) {
		options.onError = cb
	}
}

type Server struct {
	opts options
	ctx context.Context
	cancel context.CancelFunc
	connPool *sync.Map
	wg *sync.WaitGroup
	workerPool *WorkerPool
	netIdCounter *utils.AtomicInt64
	mux sync.Mutex //protect following var
	listeners map[net.Listener]bool
}

func NewServer(builders ...OptionBuilder) *Server {
	var opts options
	for _, optbuilder := range builders {
		optbuilder(&opts)
	}

	if opts.codec == nil {
		opts.codec = TypeLengthValueCodec{}
	}
	if opts.bufferSize <= 0 {
		opts.bufferSize = BufferSize256
	}
	if opts.workerPoolSize <= 0 {
		opts.workerPoolSize = DefaultWorkerPoolSize
	}

	s := Server {
		opts: opts,
		connPool: &sync.Map{},
		wg: &sync.WaitGroup{},
		netIdCounter: utils.NewAtomicInt64(0),
		listeners: make(map[net.Listener]bool),
		workerPool: newWorkerPool(opts.workerPoolSize),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return &s
}

func (s *Server) ConnPoolSize() int {
	var sz = 0
	s.connPool.Range(func(key, value interface{}) bool {
		sz++
		return true
	})
	return sz
}

func (s *Server) Broadcast(msg Message) {
	s.connPool.Range(func(key, value interface{}) bool {
		c := value.(*ServerConn)
		if err := c.Write(msg); err != nil {
			log.DefaultLogger.Errorf("Broadcast error %v, conn netId %d\n",
				err, c.NetId())
			return false
		}
		return true
	})
}

func (s *Server) Unicast(targetNetId int64, msg Message) error{
	conn, ok := s.connPool.Load(targetNetId)
	if ok {
		return conn.(*ServerConn).Write(msg)
	}
	return fmt.Errorf("conn id %d not found\n", targetNetId)
}

func (s *Server) GetConnById(id int64)(*ServerConn, bool) {
	v, ok := s.connPool.Load(id)
	if ok {
		return v.(*ServerConn), true
	}
	return nil, false
}

func (s *Server) Start(l net.Listener) error {
	s.mux.Lock()
	if s.listeners == nil {
		s.mux.Unlock()
		l.Close()
		return types.ErrServerClosed
	}
	s.listeners[l] = true
	s.mux.Unlock()
	defer func() {
		s.mux.Lock()
		if s.listeners != nil && s.listeners[l] {
			l.Close()
			delete(s.listeners, l)
		}
		s.mux.Unlock()
	}()

	log.DefaultLogger.Infof("Server starting, net %s addr %s\n",
		l.Addr().Network(), l.Addr().String())

	var tmpDelay time.Duration
	for {
		rawConn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tmpDelay == 0 {
					tmpDelay = 5 * time.Millisecond
				} else {
					tmpDelay *= 2
				}
				if max := 1*time.Second; tmpDelay >= max {
					tmpDelay = max
				}
				log.DefaultLogger.Errorf("Accept error %v, retrying in %d\n",
					err, tmpDelay)
				select {
				case <-time.After(tmpDelay):
				case <-s.ctx.Done():
				}
				continue
			}
			return err

		}
		tmpDelay = 0

		curPoolSize := s.ConnPoolSize()
		if curPoolSize >= MaxConnections {
			log.DefaultLogger.Errorf("Current conn pool over MaxConnections %d, refuse serve\n",
				curPoolSize)
			rawConn.Close()
			continue
		}

		if s.opts.tlsConfig != nil {
			rawConn = tls.Server(rawConn, s.opts.tlsConfig)
		}
		netId := s.netIdCounter.GetAndIncrement()
		sc := NewServerConn(netId, s, rawConn)
		sc.SetName(sc.rawConn.RemoteAddr().String())

		s.connPool.Store(netId, sc)

		s.wg.Add(1)
		go func() {
			sc.Start()
		}()

		log.DefaultLogger.Infof("Accept client %s, id %d, current total connSize %d\n",
			sc.Name(), netId, s.ConnPoolSize())
		s.connPool.Range(func(key, value interface{}) bool {
			i := key.(int64)
			c := value.(*ServerConn)
			log.DefaultLogger.Infof("client(%d) %s\n", i, c.Name())
			return true
		})
	}
}

func (s *Server) Stop() {
	s.mux.Lock()
	allListeners := s.listeners
	s.listeners = nil
	s.mux.Unlock()

	for l := range allListeners {
		l.Close()
		log.DefaultLogger.Infof("Stop accepting at address %s\n", l.Addr().String())
	}

	conns := map[int64]*ServerConn{}
	s.connPool.Range(func(key, value interface{}) bool {
		netId := key.(int64)
		sc := value.(*ServerConn)
		conns[netId] = sc
		return true
	})

	s.connPool = nil

	for _,sc := range conns {
		sc.rawConn.Close()
		log.DefaultLogger.Debugf("Close client %s\n", sc.Name())
	}

	s.workerPool.Close()

	s.cancel()

	s.wg.Wait()
	log.DefaultLogger.Infof("Server stopped gracefully\n")
	os.Exit(0)
}

// LoadTLSConfig returns a TLS configuration with the specified cert and key file.
func LoadTLSConfig(certFile, keyFile string, isSkipVerify bool) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: isSkipVerify,
		CipherSuites: []uint16{
			tls.TLS_RSA_WITH_RC4_128_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
	}
	now := time.Now()
	config.Time = func() time.Time { return now }
	config.Rand = rand.Reader
	return config, nil
}