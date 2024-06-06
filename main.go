package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"

	pb "github.com/ruziba3vich/realtime/genproto"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type User struct {
	ID   string
	Conn *websocket.Conn
}

type Server struct {
	Users map[string]*User
	Mutex sync.Mutex
	pb.UnimplementedMessageServiceServer
}

func NewServer() *Server {
	return &Server{
		Users: make(map[string]*User),
	}
}

func (s *Server) SendMessage(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResponse, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	targetUser, exists := s.Users[req.To]
	if !exists {
		return &pb.MessageResponse{Status: "User not found"}, nil
	}

	err := targetUser.Conn.WriteJSON(map[string]string{"from": req.From, "message": req.Message})
	if err != nil {
		return &pb.MessageResponse{Status: "Failed to send message"}, err
	}
	return &pb.MessageResponse{Status: "Message sent successfully"}, nil
}

func (s *Server) registerUserWebSocket(id string, conn *websocket.Conn) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Users[id] = &User{
		ID:   id,
		Conn: conn,
	}
}

func (s *Server) removeUser(id string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	delete(s.Users, id)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	s.registerUserWebSocket(id, conn)
	defer s.removeUser(id)

	log.Println("New connection from:", id)

	for {
		var msg map[string]string
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("Unexpected close error:", err)
			}
			break
		}

		targetID, message := msg["target"], msg["message"]
		targetUser, exists := s.Users[targetID]
		if exists {
			err = targetUser.Conn.WriteJSON(map[string]string{"from": id, "message": message})
			if err != nil {
				log.Println("Write error:", err)
				break
			}
		} else {
			conn.WriteJSON(map[string]string{"error": "User not found"})
		}
	}
}

func main() {
	server := NewServer()

	go func() {
		listener, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Failed to listen on port 50051: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterMessageServiceServer(grpcServer, server)
		reflection.Register(grpcServer)

		log.Println("Starting gRPC server on port :50051")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	http.HandleFunc("/ws", server.handleWS)
	log.Println("Starting WebSocket server on :2004")
	if err := http.ListenAndServe(":2004", nil); err != nil {
		log.Fatal(err)
	}
}
