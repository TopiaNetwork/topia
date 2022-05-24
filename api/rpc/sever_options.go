package rpc

import (
	"github.com/coocood/freecache"
)

type Options struct {
	Auth  AuthObject
	cache *freecache.Cache
}

type Option func(options *Options)

type AuthObject struct {
	// level => token
	tokenArr []string
}

func NewAuthObject(arr []string) *AuthObject {
	return &AuthObject{
		tokenArr: arr,
	}
}

func (auth *AuthObject) Token(level int) (token string, err error) {
	if level < 0 || level >= len(auth.tokenArr) {
		err = ErrAuthLevel
	}
	token = auth.tokenArr[level]
	return token, err
}

func (auth *AuthObject) Level(token string) (level int) {
	level = 0
	for i := range auth.tokenArr {
		if auth.tokenArr[i] == token {
			level = i
		}
	}
	return
}

func defaultOptions() *Options {
	return &Options{}
}

func SetAUTH(auth AuthObject) Option {
	return func(options *Options) {
		options.Auth = auth
	}
}

func SetCache(cache *freecache.Cache) Option {
	return func(options *Options) {
		options.cache = cache
	}
}
