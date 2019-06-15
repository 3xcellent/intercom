package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/3xcellent/intercom/proto"
	"google.golang.org/grpc"
)

type intercomServer struct {
	currentBroadcastBytes []byte
	currentBroadcastImgSync sync.Mutex
	lastBroadcastReceived time.Time
	hasIncomingBroadcast bool
	lastBroadcast proto.Broadcast
}

func (s *intercomServer) isCurrentlyBroadcasting() bool {
	if !s.hasIncomingBroadcast {
		return false
	}
	if time.Now().After(s.lastBroadcastReceived.Add(500 * time.Millisecond)) {
		s.hasIncomingBroadcast = false
		return false
	}
	return true
}

func (s *intercomServer) Connect(stream proto.Intercom_ConnectServer) error {
	log.Println("new stream connection established")
	ctx := stream.Context()

	go func() {
		for {
			// exit if context is done
			// or continue
			select {
			case <-ctx.Done():
				fmt.Println("outgoing stream closed: " + ctx.Err().Error())
				return
			default:
			}

			if !s.isCurrentlyBroadcasting() {
				continue
			}

			if err := stream.Send(&s.lastBroadcast); err != nil {
				fmt.Printf("send error %v", err)
			}
		}
	}()

	go func() {
		for {
			// exit if context is done
			// or continue
			select {
			case <-ctx.Done():
				fmt.Println("incoming stream closed: " + ctx.Err().Error())
				return
			default:
			}

			broadcast, err := stream.Recv()
			if err == io.EOF {
				// return will close stream from server side
				fmt.Printf("connection closed: %v\n", err)
				break
			}
			if err != nil {
				fmt.Printf("receive error %v\n", err)
				break
			}

			s.hasIncomingBroadcast = true
			s.lastBroadcastReceived = time.Now()
			s.lastBroadcast = *broadcast
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("stream closed: " + ctx.Err().Error())
			return nil
		default:
		}
	}
}

func main() {
	// create listener
	l, err := net.Listen("tcp", ":6000")
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterIntercomServer(grpcServer, &intercomServer{})

	fmt.Println("Listening on tcp://localhost:6000")

	if err := grpcServer.Serve(l); err != nil {
		panic(err)
	}
}