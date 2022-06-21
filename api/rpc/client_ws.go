package rpc

import (
	"bytes"
	"crypto/tls"
	"log"
	"net/http"
	"strconv"
	"time"

	tlog "github.com/TopiaNetwork/topia/log"
	"github.com/gorilla/websocket"
)

type WebsocketClient struct {
	addr           string
	conn           *websocket.Conn
	send           chan []byte
	receive        chan []byte
	maxMessageSize int
	pingWait       string //time
	tlsConfig      *tls.Config
	logger         tlog.Logger
	requestRes     map[string]chan []byte
}

func (ws *WebsocketClient) readPump() {
	conn := ws.conn
	defer func() {
		log.Print("readPump end")
		// ws.Close()
	}()
	pingWait, err := time.ParseDuration(ws.pingWait)
	if err != nil {
		ws.logger.Error(err.Error())
		return
	}
	conn.SetReadLimit(int64(ws.maxMessageSize))
	conn.SetReadDeadline(time.Now().Add(pingWait))
	conn.SetPingHandler(func(string) error {
		log.Print("receive ping")
		conn.SetReadDeadline(time.Now().Add(pingWait))
		err := conn.WriteMessage(websocket.PongMessage, []byte{})
		return err
	})
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			ws.logger.Error(err.Error())
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// ws.receive <- message
		go func() {
			ws.dealMessage(message)
		}()
	}
}

func (ws *WebsocketClient) writePump() {
	defer func() {
		log.Print("writePump end")
		// ws.Close()
	}()
	// send := ws.send
	conn := ws.conn
	for {
		message, ok := <-ws.send
		if !ok {
			log.Print("close connection")
			conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		w, err := conn.NextWriter(websocket.TextMessage)
		if err != nil {
			ws.logger.Error(err.Error())
			return
		}
		w.Write(message)
		if err := w.Close(); err != nil {
			ws.logger.Error(err.Error())
			return
		}
	}
}

func (ws *WebsocketClient) dealMessage(message []byte) {
	decodedMessage, err := DecodeMessage(message)
	if err != nil {
		ws.logger.Error(err.Error())
	}
	reqId := decodedMessage.RequestId
	resChan, ok := ws.requestRes[reqId]
	if !ok {
		ws.logger.Error(reqId + " res chan not found")
	}
	resChan <- decodedMessage.Payload
}

func (ws *WebsocketClient) Connect() {
	requestId, _ := DistributedID()
	dialer := &websocket.Dialer{
		TLSClientConfig: ws.tlsConfig,
	}
	var header http.Header = make(http.Header)
	header.Add("Content-Type", "text/xml; charset=UTF-8")
	header.Add("requestId", requestId)
	header.Add("maxMessageSize", strconv.Itoa(ws.maxMessageSize))
	header.Add("pingWait", ws.pingWait)
	conn, _, err := dialer.Dial(ws.addr, header)
	if err != nil {
		ws.logger.Error(err.Error())
	}
	// // ws.mutex.Lock()
	// // defer ws.mutex.Unlock()
	// if !ws.isClosed {
	// 	return
	// }
	// ws.isClosed = false
	ws.conn = conn
	ws.send = make(chan []byte)
}

func (ws *WebsocketClient) Run() {
	ws.Connect()
	go ws.writePump()
	go ws.readPump()
}

// func (ws *WebsocketClient) Close() {
// 	ws.mutex.Lock()
// 	defer ws.mutex.Unlock()
// 	if ws.isClosed {
// 		return
// 	}
// 	ws.logger.Error("close connect")
// 	ws.isClosed = true
// 	ws.conn.Close()
// 	close(ws.send)
// }
