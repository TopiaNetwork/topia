package rpc

import (
	"github.com/coocood/freecache"
	"github.com/gorilla/websocket"
	"sync"
)

type Options struct {
	auth             AuthObject
	cache            *freecache.Cache
	upgrader         *websocket.Upgrader
	websocketServers sync.Map
}

type Option func(options *Options)

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
		options.cache = cache
	}
}

func SetWebsocket(upgrader *websocket.Upgrader) Option {
	return func(options *Options) {
		options.upgrader = upgrader
	}
}
