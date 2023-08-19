package main

import (
	pb "bekarir-backend/grpc_server/pubsubpb" // Ganti dengan package name yang sesuai
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type pubSubServer struct {
	pb.UnimplementedPubSubServer
	subscribers map[chan<- *pb.Message]bool
	mu          sync.Mutex
}

func (s *pubSubServer) Subscribe(req *pb.SubscriptionRequest, stream pb.PubSub_SubscribeServer) error {
	messageChannel := make(chan *pb.Message)
	s.mu.Lock()
	s.subscribers[messageChannel] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.subscribers, messageChannel)
		close(messageChannel)
		s.mu.Unlock()
	}()

	for {
		select {
		case msg := <-messageChannel:
			if err := stream.Send(msg); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

func (s *pubSubServer) Publish(ctx context.Context, msg *pb.Message) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for ch := range s.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}

	return &pb.Empty{}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	server := grpc.NewServer()
	pubSub := &pubSubServer{
		subscribers: make(map[chan<- *pb.Message]bool),
	}
	pb.RegisterPubSubServer(server, pubSub)

	fmt.Println("gRPC Pub/Sub server started on :50051")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
