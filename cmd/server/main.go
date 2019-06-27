package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/3xcellent/intercom/proto"
	"google.golang.org/grpc"
)

type intercomServer struct {
	lastBroadcastReceived time.Time
	hasIncomingBroadcast  bool
	currentBroadcastImage proto.Image
	currentBroadcastAudio proto.Audio
}

func (s *intercomServer) isCurrentlyBroadcasting() bool {
	if !s.hasIncomingBroadcast {
		return false
	}
	if time.Now().After(s.lastBroadcastReceived.Add(300 * time.Millisecond)) {
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

			broadcast := proto.Broadcast{
				BroadcastType: &proto.Broadcast_Image{
					Image: &s.currentBroadcastImage,
				},
			}

			if err := stream.Send(&broadcast); err != nil {
				fmt.Printf("send error %v", err)
			}

			time.Sleep(time.Second / 30)
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
			image := broadcast.GetImage()
			if image != nil {
				s.currentBroadcastImage = *image
				continue
			}

			audio := broadcast.GetAudio()
			if audio != nil {
				broadcast := proto.Broadcast{
					BroadcastType: &proto.Broadcast_Audio{
						Audio: audio,
					},
				}

				if err := stream.Send(&broadcast); err != nil {
					fmt.Printf("send error %v", err)
				}
				continue
			}
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
