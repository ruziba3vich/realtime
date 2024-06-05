package main

import (
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

func main() {

}
