package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	id uuid.UUID
	server *Server
	conn *websocket.Conn
	send chan []byte
}

const (
	writeWait = 10 * time.Second
	pingPeriod = (pongWait * 9) / 10
	pongWait = 60 * time.Second;
	maxMessageSize = 512
)
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

func serveWS(server *Server, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
	 if err != nil {
        log.Println(err)
        return
    }
	id := uuid.New()
	client := &Client{
		id : id,
		server: server,
		conn : conn,
		send : make(chan []byte),
	}

	client.server.register <- client

	go client.writePump()
	go client.readPump()


}

func (c *Client) readPump() {
	defer func(){
		c.conn.Close()
		c.server.unregister <- c
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(appData string) error {
	 	c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, text, err := c.conn.ReadMessage()
		log.Printf("Message : %v" , string(text))
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		

		msg := &WSMessage{}

		reader := bytes.NewReader(text)
		decoder := json.NewDecoder(reader)
		err = decoder.Decode(msg)
		if err != nil {
			log.Printf("err: %v"  , err)
		}

		c.server.broadcast <- &Message{ClientID: c.id , Text : string(text)}

	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func(){
		c.conn.Close()
	}()
	for {
		select {
		case msg , ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		w, err := c.conn.NextWriter((websocket.TextMessage))
		if err != nil {
			return
		}
		
		w.Write(msg)
		n := len(c.send)
		for i :=0 ; i <n ; i++ {
			w.Write(msg)
		}
		if err := w.Close(); err != nil {
			return
		}
	case <- ticker.C:
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
	}
}