package main

import (
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"io"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/3xcellent/intercom/proto"
)

const (
	screenWidth  = 1280/2
	screenHeight = 720/2
	matType = gocv.MatTypeCV8UC3
)

type intercomServer struct {
	clients []string
	currentBroadcastName string
	currentBroadcastImg gocv.Mat
	lastBroadcastReceived time.Time
}

func (s *intercomServer) ClientBroadcast(stream proto.Intercom_ClientBroadcastServer) error {
	log.Println("start new ClientBroadcast")
	ctx := stream.Context()

	for {
		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// receive data from stream
		broadcast, err := stream.Recv()
		if err == io.EOF {
			// return will close stream from server side
			log.Printf("ClientBroadcast end: %s\n", s.currentBroadcastName)
			return nil
		}
		if err != nil {
			log.Printf("receive error %v\n", err)
			continue
		}

		resp := proto.ClientBroadcastResp{}

		if 	s.isCurrentlyBroadcasting() && s.currentBroadcastName != broadcast.Name {
			resp.BroadcastAccepted = false
			resp.Reason = "BACKOFF"
			resp.Status = 1
		} else {
			resp.BroadcastAccepted = true
			s.lastBroadcastReceived = time.Now()
			s.currentBroadcastName = broadcast.Name

			// update broadcastImg and send it to stream
			s.currentBroadcastImg, err = gocv.NewMatFromBytes(int(broadcast.Height), int(broadcast.Width), gocv.MatType(broadcast.Type), broadcast.Bytes)
			if err != nil {
				log.Printf("cannot create NewMatFromBytes: %v\n", err)
				continue
			}
		}

		if err := stream.Send(&resp); err != nil {
			log.Printf("ClientBroadcastResp send error %v\n", err)
		}
	}
}

func (s *intercomServer) isCurrentlyBroadcasting() bool {
	return s.lastBroadcastReceived.Add(time.Second).After(time.Now())
}

func (s *intercomServer) ServerBroadcast(stream proto.Intercom_ServerBroadcastServer) error {
	log.Println("new connection")
	ctx := stream.Context()

	for {
		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// receive data from stream
		_, err := stream.Recv()
		if err == io.EOF {
			// return will close stream from server side
			log.Println("exiting stream...")
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}

		resp := proto.ServerBroadcastResp{
			IsCurrentlyBroadcasting: s.isCurrentlyBroadcasting(),
		}
		if resp.IsCurrentlyBroadcasting {
			img := &s.currentBroadcastImg

			resp.Name = s.currentBroadcastName
			resp.Bytes = img.ToBytes()
			resp.Height = int32(img.Size()[0])
			resp.Width = int32(img.Size()[1])
			resp.Type = int32(img.Type())
		}

		if err := stream.Send(&resp); err != nil {
			log.Printf("send error %v", err)
		}
	}
}

func main() {
	// create listener
	filename := os.Args[1]

	// prepare displayImg
	bImg := gocv.NewMatWithSize(screenHeight, screenWidth, matType)
	getSizedBroadcastImg(filename, &bImg)

	l, err := net.Listen("tcp", ":6000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterIntercomServer(grpcServer, &intercomServer{})

	log.Println("Listening on tcp://localhost:6000")

	if err := grpcServer.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}


func getSizedBroadcastImg(filename string, img *gocv.Mat) {
	defaultImg := gocv.IMRead(filename, gocv.IMReadColor)
	defer defaultImg.Close()

	if defaultImg.Empty() {
		fmt.Printf("Error reading image from: %v\n", filename)
		return
	} else {
		fmt.Printf("Opening image from: %v | %#v\n", filename, defaultImg.Size())
	}
	gocv.Resize(defaultImg, img, image.Point{X: screenWidth, Y: screenHeight}, 0, 0, gocv.InterpolationDefault)
}

