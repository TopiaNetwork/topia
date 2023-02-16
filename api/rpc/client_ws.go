package rpc

import (
	"crypto/tls"
	tlog "github.com/TopiaNetwork/topia/log"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
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
	requestRes     map[string]chan *Message

	subsMsg map[clientSubscription]chan []byte // For receiving subscribed msg

}

type clientSubscription struct {
	eventName string
	subID     int
}

func (ws *WebsocketClient) readPump() {
	conn := ws.conn
	defer func() {
		log.Print("readPump end")
		// ws.Close()
	}()
	//pingWait, err := time.ParseDuration(ws.pingWait) // TODO
	//if err != nil {
	//	ws.logger.Error(err.Error())
	//	return
	//}
	conn.SetReadLimit(int64(ws.maxMessageSize))
	//conn.SetReadDeadline(time.Now().Add(pingWait)) // TODO
	//conn.SetPingHandler(func(string) error {
	//	log.Print("receive ping")
	//	conn.SetReadDeadline(time.Now().Add(pingWait))
	//	err := conn.WriteMessage(websocket.PongMessage, []byte{})
	//	return err
	//})
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			ws.logger.Error("ReadMessage from conn err: " + err.Error())
			break
		}
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
		w, err := conn.NextWriter(websocket.BinaryMessage)
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
		return
	}

	switch decodedMessage.MsgType {
	case MsgCallResp:
		resChan, ok := ws.requestRes[decodedMessage.RequestId]
		if !ok {
			ws.logger.Error(decodedMessage.RequestId + " res chan not found")
			return
		}
		resChan <- decodedMessage
	case MsgSubscribe:
		subID, err := strconv.Atoi(decodedMessage.RequestId)
		if err != nil {
			ws.logger.Error("string to int err: " + err.Error())
			return
		}
		clientSub := clientSubscription{
			eventName: decodedMessage.MethodName,
			subID:     subID,
		}
		subMsgChan, ok := ws.subsMsg[clientSub]
		if !ok {
			ws.logger.Infof("no such subscription: eventName:%v subID:%v", clientSub.eventName, clientSub.subID)
			return
		}
		subMsgChan <- decodedMessage.Payload
	default:

	}

}

func (ws *WebsocketClient) Connect() error {
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
		return err
	}
	// // ws.mutex.Lock()
	// // defer ws.mutex.Unlock()
	// if !ws.isClosed {
	// 	return
	// }
	// ws.isClosed = false
	ws.conn = conn
	return nil
}

func (ws *WebsocketClient) Run() error {
	err := ws.Connect()
	if err != nil {
		return err
	}
	go ws.writePump()
	go ws.readPump()
	return nil
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
