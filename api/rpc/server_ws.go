package rpc

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketServer struct {
	remoteAddr     string
	conn           *websocket.Conn
	maxMessageSize int64
	pongWait       time.Duration
	pingPeriod     time.Duration
	writeWait      time.Duration
	send           chan []byte
	receive        chan []byte
	isConnected    bool
	mu             sync.RWMutex
}

func (ws *WebsocketServer) ReadMessage() (messageType int, message []byte, err error) {
	err = ErrNotConnected
	if ws.IsConnected() {
		messageType, message, err = ws.conn.ReadMessage()
		if err != nil {
			ws.Close()
			return
		}
		return messageType, message, nil
	}
	return
}

func (ws *WebsocketServer) WriteMessage(messageType int, data []byte) error {
	err := ErrNotConnected
	if ws.IsConnected() {
		ws.mu.Lock()
		err = ws.conn.WriteMessage(messageType, data)
		ws.mu.Unlock()
		if err != nil {
			ws.Close()
		}
	}
	return err
}

func (ws *WebsocketServer) IsConnected() bool {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.isConnected
}

func (ws *WebsocketServer) Close() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.isConnected = false
	ws.conn.Close()
}

func (ws *WebsocketServer) reconnect(conn *websocket.Conn) {
	// ws.conn.Close()
	ws.conn = conn
}

// readPump constantly read msg from connection to receiveChan
func (ws *WebsocketServer) readPump() {
	defer func() {
		log.Print("unregister")
		close(ws.receive) // TODO 这里不确定 需要考虑关闭的流程,重复关闭会panic
	}()
	ws.conn.SetReadLimit(ws.maxMessageSize)
	//ws.conn.SetReadDeadline(time.Now().Add(ws.pongWait)) // TODO
	//ws.conn.SetPongHandler(func(string) error {
	//	log.Print("receive pong")
	//	ws.conn.SetReadDeadline(time.Now().Add(ws.pongWait))
	//	return nil
	//})
	//ws.conn.SetPingHandler(func(string) error {
	//	log.Print("receive ping")
	//	// conn.SetReadDeadline(time.Now().Add(ws.pongWait))
	//	return nil
	//})

	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		ws.receive <- message
	}
}

func (ws *WebsocketServer) writePump() {
	conn := ws.conn
	//ticker := time.NewTicker(ws.pingPeriod) // TODO
	defer func() {
		//ticker.Stop() // todo
		conn.Close()
	}()
	for {
		select {
		case message, ok := <-ws.send:
			conn.SetWriteDeadline(time.Now().Add(ws.writeWait))
			if !ok {
				// The hub closed the channel.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("write error: %v", err)
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message.
			n := len(ws.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-ws.send)
			}

			if err := w.Close(); err != nil {
				log.Printf("error: %v", err)
				return
			}
			//case <-ticker.C: // TODO
			//	conn.SetWriteDeadline(time.Now().Add(ws.writeWait))
			//	log.Print("ping")
			//	if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			//		return
			//	}
		}
	}
}

// type ServerHub struct {
// 	// Registered WebsocketServers.
// 	websocketServers sync.Map

// 	// Inbound messages from the WebsocketServers.
// 	broadcast chan []byte

// 	// Register requests from the WebsocketServers.
// 	register chan map[string]*WebsocketServer

// 	// Unregister requests from WebsocketServers.
// 	unregister chan map[string]*WebsocketServer
// }

// func NewServerHub() *ServerHub {
// 	return &ServerHub{
// 		broadcast:  make(chan []byte),
// 		register:   make(chan map[string]*WebsocketServer),
// 		unregister: make(chan map[string]*WebsocketServer),
// 	}
// }
