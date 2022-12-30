package rpc

import (
	"crypto/tls"
	"github.com/TopiaNetwork/topia/log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gregjones/httpcache"
)

type ClientOptions struct {
	timeout   time.Duration
	AUTH      string
	attempts  int
	sleepTime time.Duration
	tlsConfig *tls.Config
	cache     httpcache.Cache
	ws        *WebsocketClient
}

type ClientOption func(options *ClientOptions)

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{
		timeout:   2 * time.Second,
		AUTH:      "",
		attempts:  3,
		sleepTime: 3 * time.Second,
	}
}

func SetClientAUTH(token string) ClientOption {
	return func(options *ClientOptions) {
		options.AUTH = token
	}
}

func SetClientTimeOut(timeout time.Duration) ClientOption {
	return func(options *ClientOptions) {
		options.timeout = timeout
	}
}

func SetClientAttempts(attempts int) ClientOption {
	return func(options *ClientOptions) {
		options.attempts = attempts
	}
}

func SetClientSleepTime(sleepTime time.Duration) ClientOption {
	return func(options *ClientOptions) {
		options.sleepTime = sleepTime
	}
}

func SetClientTLS(config *tls.Config) ClientOption {
	return func(options *ClientOptions) {
		options.tlsConfig = config
	}
}

func SetClientCache(cache httpcache.Cache) ClientOption {
	return func(options *ClientOptions) {
		options.cache = cache
	}
}

// using websocket
func SetClientWS(maxMessageSize int, pingWait string, logger log.Logger) ClientOption {
	return func(options *ClientOptions) {
		options.ws = &WebsocketClient{
			maxMessageSize: maxMessageSize,
			pingWait:       pingWait,
			conn:           new(websocket.Conn),
			send:           make(chan []byte, 1),
			receive:        make(chan []byte),
			// isClosed:       true,
			requestRes: make(map[string]chan *Message),
			// mutex:          new(sync.Mutex),
			logger: logger,
		}
	}
}
