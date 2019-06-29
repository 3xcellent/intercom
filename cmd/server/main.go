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
	imgMutex                   sync.Mutex
	audioMutex                 sync.Mutex
	lastBroadcastImageReceived time.Time
	lastBroadcastAudioReceived time.Time
	hasIncomingBroadcastImage  bool
	hasIncomingBroadcastAudio  bool
	currentBroadcastImage      proto.Image
	currentBroadcastAudio      proto.Audio
}

func (s *intercomServer) isCurrentlyBroadcasting() bool {
	if !s.hasIncomingBroadcastImage && !s.hasIncomingBroadcastAudio {
		return false
	}
	if time.Now().After(s.lastBroadcastImageReceived.Add(300*time.Millisecond)) && time.Now().After(s.lastBroadcastAudioReceived.Add(300*time.Millisecond)) {
		s.hasIncomingBroadcastImage = false
		s.hasIncomingBroadcastAudio = false
		return false
	}
	return true
}

func (s *intercomServer) Connect(stream proto.Intercom_ConnectServer) error {
	log.Println("new stream connection established")
	ctx := stream.Context()

	var streamLastImageSent time.Time
	var streamLastAudioSent time.Time

	go func() {
		// SEND LOOP
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

			if time.Now().After(streamLastImageSent.Add(1 / 30 * time.Millisecond)) {
				broadcast.BroadcastType = &proto.Broadcast_Image{
					Image: &s.currentBroadcastImage,
				}

				s.imgMutex.Lock()
				if err := stream.Send(&broadcast); err != nil {
					fmt.Printf("send error %v", err)
				}
				fmt.Print("A")
				s.imgMutex.Unlock()
			}

			if streamLastAudioSent != s.lastBroadcastAudioReceived {
				broadcast.BroadcastType = &proto.Broadcast_Audio{
					Audio: &s.currentBroadcastAudio,
				}

				s.audioMutex.Lock()
				if err := stream.Send(&broadcast); err != nil {
					fmt.Printf("send error %v", err)
				}
				streamLastAudioSent = s.lastBroadcastAudioReceived
				fmt.Print("I")
				s.audioMutex.Unlock()
			}
		}
	}()

	// RECEIVE LOOP
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
				fmt.Print("i")
				s.imgMutex.Lock()
				s.currentBroadcastImage = *image
				s.imgMutex.Unlock()

				s.hasIncomingBroadcastImage = true
				s.lastBroadcastImageReceived = time.Now()
				continue
			}

			audio := broadcast.GetAudio()
			if audio != nil {
				fmt.Print("a")
				s.audioMutex.Lock()
				s.currentBroadcastAudio = *audio
				s.audioMutex.Unlock()

				s.hasIncomingBroadcastAudio = true
				s.lastBroadcastAudioReceived = time.Now()
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
