package rpc

import (
	"bytes"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	tlog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	"github.com/gregjones/httpcache"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Client struct {
	addr    string
	options *ClientOptions
	logger  tlog.Logger
}

// NewClient creates a new client
func NewClient(addr string, options ...ClientOption) (*Client, error) {
	logger, err := tlog.CreateMainLogger(logcomm.DebugLevel, tlog.JSONFormat, tlog.StdErrOutput, "")
	if err != nil {
		panic(err)
	}
	c := &Client{
		addr:    addr,
		options: defaultClientOptions(),
		logger:  logger,
	}
	for _, fn := range options {
		fn(c.options)
	}

	if c.options.ws != nil {
		addr := strings.Trim("ws://"+addr+"/websocket", " ")
		c.options.ws.addr = addr
		c.options.ws.Run()
		// c.options.recws.Dial(addr, nil)
		// go func() {
		// 	timeSleep := time.Duration(0)
		// 	timeIncrease := 500 * time.Microsecond
		// 	for {
		// 		_, data, err := c.options.recws.ReadMessage()
		// 		if err != nil {
		// 			timeSleep := timeSleep + timeIncrease
		// 			log.Print(err.Error())
		// 			time.Sleep(timeSleep)
		// 			continue
		// 		}
		// 		message, _ := DecodeMessage(data)
		// 		receive, ok := c.requestRes[message.RequestId]
		// 		if !ok {
		// 			continue
		// 		}
		// 		timeSleep = time.Duration(0)
		// 		receive <- message.Payload
		// 	}
		// }()

		// go func() {
		// 	for {
		// 		data := <-c.send
		// 		err := c.options.recws.WriteMessage(1, data)
		// 		if err!=nil {

		// 		}
		// 	}
		// }()
	}

	return c, nil
}

func (c *Client) sendPostRetry(postUrl string, reqBody []byte) (*Message, error) {
	if c.options.attempts <= 0 {
		return c.sendPost(postUrl, reqBody)
	}
	errString := ""
	for index := 0; index < c.options.attempts; index++ {
		res, err := c.sendPost(postUrl, reqBody)
		if err == nil {
			return res, nil
		}
		if index != 0 {
			errString += "|" + err.Error()
		} else {
			errString += err.Error()
		}
		time.Sleep(c.options.sleepTime * time.Duration(2*index+1))
	}

	return nil, errors.New("SendRetry err:" + errString)
}

func (c *Client) sendPost(postUrl string, reqBody []byte) (*Message, error) {
	client := &http.Client{
		Transport: &httpcache.Transport{
			Cache:               c.options.cache,
			MarkCachedResponses: true,
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					conn, err := net.DialTimeout(netw, addr, c.options.timeout)
					if err != nil {
						return nil, err
					}
					_ = conn.SetDeadline(time.Now().Add(c.options.timeout))
					return conn, nil
				},
				TLSClientConfig:       c.options.tlsConfig,
				ResponseHeaderTimeout: c.options.timeout,
			},
		},
	}
	requestDo, err := http.NewRequest("POST", postUrl, bytes.NewReader(reqBody))
	requestDo.Header.Set("Content-Type", "text/xml; charset=UTF-8")
	if err != nil {
		c.logger.Errorf("NewRequest error: %v", err)
		return nil, errors.New("httpPost err: " + err.Error())
	}
	res, err := client.Do(requestDo)
	if nil != err {
		c.logger.Errorf("httpPost error: %v, url: %v, params: %v\n", err, postUrl, string(reqBody))
		return nil, errors.New("httpPost err: " + err.Error())
	}
	defer res.Body.Close()

	message, err := IODecodeMessage(res.Body)
	if nil != err {
		c.logger.Errorf("IODecodeMessage err: %v", err)
		return nil, errors.New("IODecodeMessage err: " + err.Error())
	}
	return message, nil
}

func (c *Client) Call(methodName string, inArgs ...interface{}) (res *Message, err error) {
	var payload []byte
	requestId, err := DistributedID()
	if err != nil {
		return nil, err
	}

	if len(inArgs) != 0 {
		payload, err = Encode(inArgs)
		if err != nil {
			return nil, err
		}
	}
	data, err := EncodeMessage(requestId, methodName, c.options.AUTH, &ErrMsg{}, payload)
	if err != nil {
		return nil, err
	}
	url := strings.Trim("http://"+c.addr+"/"+methodName+"/", " ")
	return c.sendPostRetry(url, data)
}

func (c *Client) CallWithWS(methodName string, inArgs ...interface{}) (requestId string, res chan []byte, err error) {
	var payload []byte
	requestId, err = DistributedID()
	log.Print(requestId)
	if err != nil {
		return "", nil, err
	}

	if len(inArgs) != 0 {
		payload, _ = Encode(inArgs)
	}
	data, _ := EncodeMessage(requestId, methodName, c.options.AUTH, &ErrMsg{}, payload)
	res = make(chan []byte)
	c.options.ws.requestRes[requestId] = res
	c.options.ws.send <- data
	// c.options.recws.WriteMessage(websocket.TextMessage, data)
	return
}

// func (c *Client) Test() {
// 	c.options.recws.Conn.Close()
// }
