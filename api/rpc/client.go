package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
	serverURL  string
	scheme     string // transport protocol: one of http/https/ws/wss
	serverHost string
	options    *ClientOptions
	logger     tlog.Logger
}

// NewClient creates a new client
func NewClient(addr string, options ...ClientOption) (*Client, error) {
	logger, err := tlog.CreateMainLogger(logcomm.DebugLevel, tlog.JSONFormat, tlog.StdErrOutput, "")
	if err != nil {
		panic(err)
	}

	parsedURL, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme != "http" &&
		parsedURL.Scheme != "https" &&
		parsedURL.Scheme != "ws" &&
		parsedURL.Scheme != "wss" {
		return nil, errors.New("input unsupported client url scheme")
	}

	if len(parsedURL.Host) == 0 || len(parsedURL.Port()) == 0 {
		return nil, errors.New("input illegal client url")
	}

	c := &Client{
		serverURL: fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host),
		options:   defaultClientOptions(),
		logger:    logger,
	}
	for _, fn := range options {
		fn(c.options)
	}

	c.scheme = parsedURL.Scheme
	c.serverHost = parsedURL.Host

	if parsedURL.Scheme == "https" || parsedURL.Scheme == "wss" {
		if c.options.tlsConfig == nil {
			return nil, errors.New("tls config isn't set")
		}
	}

	if parsedURL.Scheme == "ws" || parsedURL.Scheme == "wss" {
		if c.options.ws == nil {
			return nil, errors.New("cannot use websocket without websocket-settings")
		}
		c.options.ws.addr = fmt.Sprintf("%s/websocket", c.serverURL)
		c.options.ws.tlsConfig = c.options.tlsConfig
		err = c.options.ws.Run()
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) sendPostRetry(postUrl string, reqBody []byte, requestId string) (*Message, error) {
	if c.options.attempts <= 0 {
		return c.sendPost(postUrl, reqBody)
	}
	for index := 0; index < c.options.attempts; index++ {
		res, err := c.sendPost(postUrl, reqBody)
		if err != nil {
			// If got timeout err, backoff and try again.
			if e, ok := err.(net.Error); ok && e.Timeout() {
				time.Sleep(c.options.sleepTime * time.Duration(2*index+1))
				continue
			}

			// If got non-timeout err, won't retry.
			return nil, err
		}

		if res.RequestId != requestId {
			return nil, errors.New("response has incorrect requestID")
		}

		return res, nil

	}

	return nil, errors.New("retry send timeout for [" + strconv.Itoa(c.options.attempts) + "] times")
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
	if err != nil {
		c.logger.Errorf("httpPost error: %v, url: %v, params: %v\n", err, postUrl, string(reqBody))
		return nil, err
	}
	defer res.Body.Close()

	message, err := IODecodeMessage(res.Body)
	if err != nil {
		c.logger.Errorf("IODecodeMessage err: %v", err)
		return nil, errors.New("IODecodeMessage err: " + err.Error())
	}
	return message, nil
}

func (c *Client) Call(methodName string, response interface{}, inArgs ...interface{}) (err error) {
	var message *Message
	if c.scheme == "http" || c.scheme == "https" {
		message, err = c.callWithHttp(methodName, inArgs)
		if err != nil {
			return err
		}
	} else if c.scheme == "ws" || c.scheme == "wss" {
		message, err = c.callWithWS(methodName, inArgs)
		if err != nil {
			return err
		}
	} else {
		return errors.New("invalid scheme")
	}
	if message.ErrMsg.Errtype != NoErr {
		return errors.New(message.ErrMsg.ErrString)
	}

	if message.Payload != nil {
		err = json.Unmarshal(message.Payload, response)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) callWithHttp(methodName string, inArgs ...interface{}) (res *Message, err error) {
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
	data, err := EncodeMessage(MsgCall, requestId, methodName, c.options.AUTH, &ErrMsg{}, payload)
	if err != nil {
		return nil, err
	}

	postUrl := fmt.Sprintf("%s/%s/", c.serverURL, methodName)
	return c.sendPostRetry(postUrl, data, requestId)
}

func (c *Client) callWithWS(methodName string, inArgs ...interface{}) (res *Message, err error) {
	if c.options.ws == nil {
		return nil, errors.New("it is not a websocket client")
	}

	var payload []byte
	requestId, err := DistributedID()
	if err != nil {
		return nil, err
	}

	if len(inArgs) != 0 {
		payload, _ = Encode(inArgs)
	}
	data, _ := EncodeMessage(MsgCall, requestId, methodName, c.options.AUTH, &ErrMsg{}, payload)

	var respChan = make(chan *Message, 1)
	c.options.ws.requestRes[requestId] = respChan
	defer delete(c.options.ws.requestRes, requestId)
	c.options.ws.send <- data
	// c.options.recws.WriteMessage(websocket.TextMessage, data)

	select {
	case res = <-respChan:
		return res, nil

		// TODO case timeout
	}

}

// TODO 仅测试用
func (c *Client) CallWithWS(methodName string, inArgs ...interface{}) (res *Message, err error) {
	if c.options.ws == nil {
		return nil, errors.New("it is not a websocket client")
	}

	var payload []byte
	requestId, err := DistributedID()
	if err != nil {
		return nil, err
	}

	if len(inArgs) != 0 {
		payload, _ = Encode(inArgs)
	}
	data, _ := EncodeMessage(MsgCall, requestId, methodName, c.options.AUTH, &ErrMsg{}, payload)

	var respChan = make(chan *Message, 1)
	c.options.ws.requestRes[requestId] = respChan
	c.options.ws.send <- data
	// c.options.recws.WriteMessage(websocket.TextMessage, data)

	select {
	case res = <-respChan:
		delete(c.options.ws.requestRes, requestId)
		return res, nil

		// TODO case timeout
	}

}

func (c *Client) Subscribe(eventName string, filterString string) (subMsgChan <-chan []byte, subID int, err error) {
	if c.options.ws == nil {
		return nil, 0, errors.New("it is not a websocket client")
	}

	requestId, err := DistributedID()
	if err != nil {
		return nil, 0, err
	}

	inArgs := []interface{}{c.options.ws.conn.LocalAddr().String(), eventName, filterString}
	inArgsBytes, err := json.Marshal(inArgs)
	if err != nil {
		return nil, 0, err
	}

	data, _ := EncodeMessage(MsgCall, requestId, "subscriptionCenter_Subscribe", c.options.AUTH, &ErrMsg{}, inArgsBytes)

	var subStatChan = make(chan *Message, 1)
	c.options.ws.requestRes[requestId] = subStatChan
	defer delete(c.options.ws.requestRes, requestId)
	c.options.ws.send <- data

	select {
	case subStat := <-subStatChan:
		if subStat.ErrMsg.Errtype != NoErr { // fail to subscribe
			return nil, 0, errors.New(subStat.ErrMsg.ErrString)
		}

		var outArgs []interface{}

		json.Unmarshal(subStat.Payload, &outArgs)

		subID = int(outArgs[0].(float64))

		var newSubMsgChan = make(chan []byte, 1) // TODO chan缓存的大小需要确定
		var newSub = clientSubscription{
			eventName: eventName,
			subID:     subID,
		}
		c.options.ws.subsMsg[newSub] = newSubMsgChan

		return newSubMsgChan, subID, nil

		// TODO case timeout
	}
}

func (c *Client) UnSubscribe(eventName string, subID int) error {
	if c.options.ws == nil {
		return errors.New("it is not a websocket client")
	}
	if len(eventName) == 0 || subID <= 0 {
		return errors.New("illegal UnSubscribe arguments")
	}

	requestId, err := DistributedID()
	if err != nil {
		return err
	}

	inArgs := []interface{}{c.options.ws.conn.LocalAddr().String(), eventName, subID}
	inArgsBytes, err := json.Marshal(inArgs)
	if err != nil {
		return err
	}

	data, _ := EncodeMessage(MsgCall, requestId, "subscriptionCenter_UnSubscribe", c.options.AUTH, &ErrMsg{}, inArgsBytes)

	var unSubStatChan = make(chan *Message, 1)
	c.options.ws.requestRes[requestId] = unSubStatChan
	defer delete(c.options.ws.requestRes, requestId)
	c.options.ws.send <- data

	select {
	case unSubStat := <-unSubStatChan:

		if unSubStat.ErrMsg.Errtype != NoErr { // fail to subscribe
			return errors.New(unSubStat.ErrMsg.ErrString)
		}

		var subToDelete = clientSubscription{
			eventName: eventName,
			subID:     subID,
		}
		var subMsgChan = c.options.ws.subsMsg[subToDelete]

		delete(c.options.ws.subsMsg, subToDelete)
		close(subMsgChan)

		return nil

		// TODO case timeout
	}
}

func (c *Client) UnSubscribeAll() error {
	if c.options.ws == nil {
		return errors.New("it is not a websocket client")
	}

	requestId, err := DistributedID()
	if err != nil {
		return err
	}

	inArgs := []interface{}{c.options.ws.conn.LocalAddr().String()}
	inArgsBytes, err := json.Marshal(inArgs)
	if err != nil {
		return err
	}

	data, _ := EncodeMessage(MsgCall, requestId, "subscriptionCenter_UnSubscribeAll", c.options.AUTH, &ErrMsg{}, inArgsBytes)

	var unSubStatChan = make(chan *Message, 1)
	c.options.ws.requestRes[requestId] = unSubStatChan
	defer delete(c.options.ws.requestRes, requestId)
	c.options.ws.send <- data

	select {
	case unSubStat := <-unSubStatChan:
		if unSubStat.ErrMsg.Errtype != NoErr { // fail to subscribe
			return errors.New(unSubStat.ErrMsg.ErrString)
		}

		var chansToClose []chan []byte

		for k, subChan := range c.options.ws.subsMsg {
			chansToClose = append(chansToClose, subChan)
			delete(c.options.ws.subsMsg, k)
		}

		for _, chanToClose := range chansToClose {
			close(chanToClose)
		}

		return nil

		// TODO case timeout
	}
}

func (c *Client) CloseServer() error {
	requestId, err := DistributedID()
	if err != nil {
		return err
	}

	data, err := EncodeMessage(MsgCall, requestId, "CloseServer", c.options.AUTH, &ErrMsg{}, nil)
	if err != nil {
		return err
	}

	var schema = "http"
	if c.options.tlsConfig != nil {
		schema = "https"
	}

	postUrl := fmt.Sprintf("%s://%s/%s/", schema, c.serverHost, "CloseServer")
	resp, err := c.sendPostRetry(postUrl, data, requestId)
	if err != nil {
		return err
	}
	if resp.ErrMsg.Errtype != NoErr {
		return errors.New(resp.ErrMsg.ErrString)
	}

	return nil
}

//func (c *Client) Close() error {
//	if c.options.ws != nil {
//		close(c.options.ws.send)
//		return c.options.ws.conn.Close()
//	}
//	return nil
//
//}

// func (c *Client) Test() {
// 	c.options.recws.Conn.Close()
// }
