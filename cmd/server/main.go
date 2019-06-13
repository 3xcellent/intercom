package main

import (
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"io"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/3xcellent/intercom/proto"
)

const (
	screenWidth  = 1280/2
	screenHeight = 720/2
	//mirrorWindowWidth = screenWidth/4
	//mirrorWindowHeight = screenHeight/4
	//mirrorWindowX = screenHeight - mirrorWindowHeight - (mirrorWindowHeight/4)
	//mirrorWindowY = screenWidth - mirrorWindowWidth - (mirrorWindowWidth/4)
	matType = gocv.MatTypeCV8UC3
)

type intercomServer struct {
	clients []string
	currentBroadcastName string
	currentBroadcastImg gocv.Mat
	isCurrentlyBroadcasting bool
	defaultBackgroundImg gocv.Mat
}

func (s *intercomServer) ClientBroadcast(stream proto.Intercom_ClientBroadcastServer) error {
	log.Println("start new server")
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
			s.isCurrentlyBroadcasting = false
			return nil
		}
		if err != nil {
			log.Printf("receive error %v\n", err)
			continue
		}

		// process data
		s.currentBroadcastName = broadcast.Name
		s.isCurrentlyBroadcasting = true

		// update broadcastImg and send it to stream
		s.currentBroadcastImg, err = gocv.NewMatFromBytes(screenHeight, screenWidth, matType, broadcast.Bytes)
		if err != nil {
			log.Printf("cannot create NewMatFromBytes: %v\n", err)
			continue
		}

		resp := proto.ClientBroadcastResp{
			Status: 			  0,
			BroadcastAccepted:    true,
			Reason:               "",
		}

		if err := stream.Send(&resp); err != nil {
			log.Printf("ClientBroadcastResp send error %v\n", err)
		}
	}
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
		req, err := stream.Recv()
		if err == io.EOF {
			// return will close stream from server side
			log.Println("exit")
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}

		// process data
		// TODO check if already listed as a client??
		s.clients = []string{req.Name}

		resp := proto.ServerBroadcastResp{
			IsCurrentlyBroadcasting: true,
			Name:                 	s.currentBroadcastName,
			Bytes:                	s.defaultBackgroundImg.ToBytes(),
			Height:  	int32(s.defaultBackgroundImg.Size()[0]),
			Width:  	int32(s.defaultBackgroundImg.Size()[1]),
			Type:  		int32(s.defaultBackgroundImg.Type()),
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
	proto.RegisterIntercomServer(grpcServer, &intercomServer{
		defaultBackgroundImg: bImg,
	})

	log.Println("Listening on tcp://localhost:6000")

	if err := grpcServer.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}


func getSizedBroadcastImg(filename string, img *gocv.Mat) {
	defaultImg := gocv.IMRead(filename, gocv.IMReadColor)
	defer defaultImg.Close()

	if defaultImg.Empty() {
		fmt.Println("Error reading image from: %v\n", filename)
		return
	} else {
		fmt.Println("Opening image from: %v | %#v\n", filename, defaultImg.Size())
	}
	gocv.Resize(defaultImg, img, image.Point{X: screenWidth, Y: screenHeight}, 0, 0, gocv.InterpolationDefault)
}

