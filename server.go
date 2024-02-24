package main

import (
	"bytes"
	"html/template"
	"log"

	"github.com/google/uuid"
)

type Message struct {
	ClientID uuid.UUID `json:"clientID"`
	Text     string  `json:"text"`
}

// type WSMessage struct {
// 	Text    string      `json:"text"`
// 	Headers interface{} `json:"HEADERS"`
// }

type Server struct {
	clients    map[*Client]bool
	messages []*Message
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = true

			log.Printf("Client registered %s", client.id)
		case client := <-s.unregister:
			if _,ok := s.clients[client]; ok {
				log.Printf("Client unregistered %s" ,client.id)
				close(client.send)
				delete(s.clients, client)
			}
		case msg := <-s.broadcast:
			s.messages = append(s.messages,msg)
			for client := range s.clients {
				select {
				case client.send <- getMessageTemplate(msg) :
				default:
					close(client.send)
					delete(s.clients,client)
				}
			}
		}
	}
}

func getMessageTemplate(msg *Message) []byte{
	tmplt , err := template.ParseFiles("templates/message.html")
	if err != nil {
		log.Fatalf("tempalte Parsing: %s", err)
	}
	var renderedMessage bytes.Buffer
	err = tmplt.Execute(&renderedMessage,msg)
		if err != nil {
		log.Fatalf("tempalte Parsing: %s", err)
		}
		return renderedMessage.Bytes()


}