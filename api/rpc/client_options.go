package rpc

import (
	"time"
)

type ClientOptions struct {
	timeout   time.Duration
	AUTH      string
	attempts  int
	sleepTime time.Duration
}

type ClientOption func(options *ClientOptions)

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{
		timeout:   time.Second * 2,
		AUTH:      "",
		attempts:  3,
		sleepTime: 3,
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
