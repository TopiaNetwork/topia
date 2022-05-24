package rpc

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
)

type Client struct {
	addr    string
	options *ClientOptions
}

// NewClient creates a new client
func NewClient(addr string, options ...ClientOption) *Client {
	client := &Client{
		addr:    addr,
		options: defaultClientOptions(),
	}
	for _, fn := range options {
		fn(client.options)
	}
	return client
}

func (c *Client) sendPostRetry(postUrl string, reqBody []byte) ([]byte, error) {
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

func (c *Client) sendPost(postUrl string, reqBody []byte) ([]byte, error) {
	diskcache := diskcache.New("data")
	client := &http.Client{
		Transport: &httpcache.Transport{
			Cache:               diskcache,
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
				ResponseHeaderTimeout: c.options.timeout,
			},
		},
	}
	requestDo, err := http.NewRequest("POST", postUrl, bytes.NewReader(reqBody))
	requestDo.Header.Set("Content-Type", "text/xml; charset=UTF-8")
	if err != nil {
		log.Printf("NewRequest error: %v", err)
		return nil, errors.New("httpPost err:" + err.Error())
	}
	res, err := client.Do(requestDo)
	if nil != err {
		log.Printf("httpPost error: %v, url: %v, params: %v\n", err, postUrl, reqBody)
		return nil, errors.New("httpPost err:" + err.Error())
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if nil != err {
		log.Println("ReadAll err:", err)
		return nil, errors.New("ReadAll err:" + err.Error())
	}
	return data, nil
}

func (c *Client) Call(methodName string, inArgs []interface{}) (res []byte, err error) {
	requestData := make(map[string]interface{}, 2)
	requestId, err := DistributedID()
	if err != nil {
		return nil, err
	}
	requestData["Id"] = requestId
	requestData["InArgs"] = inArgs
	requestData["Auth"] = c.options.AUTH
	inBuf, err := Encode(requestData)
	if err != nil {
		return nil, err
	}
	url := strings.Trim(c.addr+"/"+methodName+"/", " ")
	return c.sendPostRetry(url, inBuf)

}
