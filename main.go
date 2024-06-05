package main

import (
	"io"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

type (
	Server struct {
		conns map[*websocket.Conn]bool
	}
)

func NewServer() *Server {
	return &Server{
		conns: make(map[*websocket.Conn]bool),
	}
}

func (s *Server) handleWS(ws *websocket.Conn) {
	log.Println("new connection has been caught :", ws.RemoteAddr())
	s.conns[ws] = true

	s.readLoop(ws)
}

func (s *Server) readLoop(ws *websocket.Conn) {
	buf := make([]byte, 2048)
	for {
		n, err := ws.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
		} else {
			msg := buf[:n]
			log.Println(string(msg))
			ws.Write([]byte("thank you for the message"))
		}
	}
}

func main() {
	server := NewServer()

	http.Handle("/ws", websocket.Handler(server.handleWS))
	http.ListenAndServe(":2004", nil)
}
