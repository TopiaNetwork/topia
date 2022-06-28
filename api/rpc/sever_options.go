package rpc

import (
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/gorilla/websocket"
)

type Options struct {
	auth             AuthObject
	cc               *freecache.Cache
	upgrader         *websocket.Upgrader
	websocketServers sync.Map
}

type Option func(options *Options)

type AuthObject struct {
	// authority => token
	tokenArr map[byte]string
}

func NewAuthObject(m map[byte]string) *AuthObject {
	return &AuthObject{
		tokenArr: m,
	}
}

func (auth *AuthObject) Token(authority byte) (token string, err error) {
	token, ok := auth.tokenArr[authority]
	if !ok {
		token = RandStringRunes(10) + time.Stamp
		auth.tokenArr[authority] = token
	}
	return token, err
}

func (auth *AuthObject) Level(token string) (authority byte) {
	for k, v := range auth.tokenArr {
		if token == v {
			return k
		}
	}
	return None
}

func defaultOptions() *Options {
	return &Options{}
}

func SetAUTH(auth AuthObject) Option {
	return func(options *Options) {
		options.auth = auth
	}
}

func SetCache(cache *freecache.Cache) Option {
	return func(options *Options) {
		options.cc = cache
	}
}

func SetWebsocket(upgrader *websocket.Upgrader) Option {
	return func(options *Options) {
		options.upgrader = upgrader
	}
}
