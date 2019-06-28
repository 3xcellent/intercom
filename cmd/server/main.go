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
	imgMutex 				   sync.Mutex
	lastBroadcastImageReceived time.Time
	lastBroadcastAudioReceived time.Time
	hasIncomingBroadcast       bool
	currentBroadcastImage      proto.Image
	currentBroadcastAudio      proto.Audio
}

func (s *intercomServer) isCurrentlyBroadcasting() bool {
	if !s.hasIncomingBroadcast {
		return false
	}
	if time.Now().After(s.lastBroadcastImageReceived.Add(300 * time.Millisecond)) {
		s.hasIncomingBroadcast = false
		return false
	}
	return true
}

func (s *intercomServer) Connect(stream proto.Intercom_ConnectServer) error {
	log.Println("new stream connection established")
	ctx := stream.Context()

	var streamLastImageSent time.Time
	//var streamLastAudioSent time.Time

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

			broadcast := proto.Broadcast{}

			s.imgMutex.Lock()
			if streamLastImageSent != s.lastBroadcastImageReceived {
				broadcast.BroadcastType = &proto.Broadcast_Image{
					Image: &s.currentBroadcastImage,
				}
				if err := stream.Send(&broadcast); err != nil {
					fmt.Printf("send error %v", err)
				}
			}
			s.imgMutex.Unlock()
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


			image := broadcast.GetImage()
			if image != nil {

				s.imgMutex.Lock()
				s.currentBroadcastImage = *image
				s.imgMutex.Unlock()

				s.hasIncomingBroadcast = true
				s.lastBroadcastImageReceived = time.Now()
				continue
			}

			audio := broadcast.GetAudio()
			if audio != nil {
				s.currentBroadcastAudio = *audio
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
