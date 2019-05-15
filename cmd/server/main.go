package main

import (
	"io"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/3xcellent/intercom/proto"
)

type intercomServer struct {
}

func (s *intercomServer) Broadcast(stream proto.Intercom_BroadcastServer) error {
	log.Println("Started stream")
	for {
		in, err := stream.Recv()
		log.Println("Received value")
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Println("Got " + in.Text)
	}
}

func main() {
	grpcServer := grpc.NewServer()
	proto.RegisterIntercomServer(grpcServer, &intercomServer{})

	l, err := net.Listen("tcp", ":6000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:6000")
	grpcServer.Serve(l)
}
