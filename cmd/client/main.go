package main

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"image"
	"io"
	"log"
	"math"
	"os"

	"github.com/3xcellent/intercom/proto"
	"gocv.io/x/gocv"
)

const (
	screenWidth  = 1280/2
	screenHeight = 720/2

	mirrorWindowWidth = screenWidth/4
	mirrorWindowHeight = screenHeight/4
	mirrorWindowX = screenHeight - mirrorWindowHeight - (mirrorWindowHeight/4)
	mirrorWindowY = screenWidth - mirrorWindowWidth - (mirrorWindowWidth/4)

	videoBroadcastWidth = screenWidth/2
	videoBroadcastHeight = screenHeight/2
	videoBroadcastX = screenHeight/2 - videoBroadcastHeight/2
	videoBroadcastY = screenWidth/2 - videoBroadcastWidth/2

	matType = gocv.MatTypeCV8UC3
)


func main() {
	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tintercom [camera ID] [path/to/background_img]")
		return
	}
	deviceID := os.Args[1]
	filename := os.Args[2]

	// prepare displayImg
	bgImg := gocv.NewMatWithSize(screenHeight, screenWidth, matType)
	getSizedBackgroundImg(filename, &bgImg)
	defer bgImg.Close()

	displayImg := bgImg.Clone()
	defer displayImg.Close()

	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	window := gocv.NewWindow("Capture Window")
	defer window.Close()

	videoPreviewImg := gocv.NewMatWithSize(mirrorWindowHeight, mirrorWindowWidth, gocv.MatTypeCV8UC3)
	defer videoPreviewImg.Close()

	videoBroadcastImg := gocv.NewMatWithSize(videoBroadcastHeight, videoBroadcastWidth, gocv.MatTypeCV8UC3)
	defer videoPreviewImg.Close()

	// dail server
	conn, err := grpc.Dial(":6000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}

	// create streams
	client := proto.NewIntercomClient(conn)
	serverBroadcastStream, err := client.ServerBroadcast(context.Background())
	if err != nil {
		log.Fatalf("openn stream error %v", err)
	}

	clientBroadcastStream, err := client.ClientBroadcast(context.Background())
	if err != nil {
		log.Fatalf("openn stream error %v", err)
	}

	receivingBroadcast := false
	isBroadcasting := false

	// main loop
	for {
		serverImg := getServerBroadcastImg(serverBroadcastStream)
		if serverImg == nil && receivingBroadcast {
			receivingBroadcast = false
			fmt.Println("incoming broadcast ended")
		}
		if !receivingBroadcast && serverImg != nil {
			receivingBroadcast = true
			fmt.Println("receiving incoming broadcast")
			fmt.Println("placing window at: %d,%d", videoBroadcastX, videoBroadcastY)

		}

		if receivingBroadcast {
			resizeVideoBroadcastImg(serverImg, &videoBroadcastImg)
			updateDisplayWithVideoBroadcast(&displayImg, videoBroadcastImg)
		}

		videoCaptureImg := getVideoCaptureImg(webcam)

		if videoCaptureImg.Empty() {
			if isBroadcasting {
				isBroadcasting = false
				fmt.Println("outgoing broadcast ended")
			}
			continue
		}

		fmt.Printf("videoCaptureImg.Size(): %v", videoCaptureImg.Size())
		if !isBroadcasting {
			isBroadcasting = true
			fmt.Println("outgoing broadcast starting")
		}

		broadcastImg(clientBroadcastStream, videoCaptureImg)
		resizeVideoPreviewImg(videoCaptureImg, &videoPreviewImg)
		updateDisplayWithVideoPreview(&displayImg, videoPreviewImg)

		window.IMShow(displayImg)
		if window.WaitKey(1) == 27 {
			break
		}
	}
}

func broadcastImg(stream proto.Intercom_ClientBroadcastClient, img gocv.Mat) error {
	req := proto.ClientBroadcastReq{
		Name:                 "dude",
		Height:               int32(img.Size()[0]),
		Width:                int32(img.Size()[1]),
		Type:                 int32(img.Type()),
		Bytes:                img.ToBytes(),
	}
	if err := stream.Send(&req); err != nil {
		log.Fatalf("can not send %v", err)
	}

	resp, err := stream.Recv()
	if err == io.EOF {
		return errors.New(io.EOF.Error())
	}
	if err != nil {
		log.Fatalf("can not receive %v", err)
	}
	if resp.BroadcastAccepted != true {
		return errors.New(resp.Reason)
	}
	return nil
}

func getServerBroadcastImg(stream proto.Intercom_ServerBroadcastClient) *gocv.Mat {
	req := proto.ServerBroadcastReq{Name: "dude"}
	if err := stream.Send(&req); err != nil {
		log.Fatalf("can not send %v", err)
	}

	resp, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		log.Fatalf("can not receive %v", err)
	}
	if !resp.GetIsCurrentlyBroadcasting() {
		return nil
	}

	bImg, err := gocv.NewMatFromBytes(int(resp.Height), int(resp.Width), gocv.MatType(resp.Type), resp.Bytes)
	if err != nil {
		log.Fatalf("can not create NewMatFromBytes %v\n", err)
		return nil
	}
	return &bImg
}

func getSizedBackgroundImg(filename string, img *gocv.Mat) {
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

func updateDisplayWithVideoBroadcast(displayImg *gocv.Mat, mirrorImg gocv.Mat) {
	for x := 0; x <= mirrorImg.Size()[0]-1; x++ {
		for y := 0; y <= videoBroadcastWidth; y++ {
			displayImg.SetIntAt3(x+videoBroadcastX, y+videoBroadcastY, 0, mirrorImg.GetIntAt3(x, y, 0))
		}
	}
}

func updateDisplayWithVideoPreview(displayImg *gocv.Mat, mirrorImg gocv.Mat) {
	for x := 0; x <= mirrorImg.Size()[0]-1; x++ {
		for y := 0; y <= mirrorWindowWidth; y++ {
			displayImg.SetIntAt3(x+mirrorWindowX, y+mirrorWindowY, 0, mirrorImg.GetIntAt3(x, mirrorWindowWidth-y, 0))
		}
	}
}

func getVideoCaptureImg(webcam *gocv.VideoCapture) gocv.Mat {
	videoCapture := gocv.NewMat()
	if ok := webcam.Read(&videoCapture); !ok {
		fmt.Println("didn't read from cam")
		return videoCapture
	}
	return videoCapture
}

func resizeVideoPreviewImg(videoCaptureImg gocv.Mat, sizedImg *gocv.Mat) {
	screenCapRatio := float64(float64(videoCaptureImg.Size()[1])/float64(videoCaptureImg.Size()[0]))
	mirrorWindowScaledHeight := int(math.Floor(mirrorWindowWidth/screenCapRatio))
	gocv.Resize(videoCaptureImg, sizedImg, image.Point{X: mirrorWindowWidth, Y: mirrorWindowScaledHeight}, 0, 0, gocv.InterpolationDefault)
}


func resizeVideoBroadcastImg(origImg *gocv.Mat, sizedImg *gocv.Mat) {
	screenCapRatio := float64(float64(origImg.Size()[1])/float64(origImg.Size()[0]))
	scaledHeight := int(math.Floor(videoBroadcastWidth/screenCapRatio))
	gocv.Resize(*origImg, sizedImg, image.Point{X: videoBroadcastWidth, Y: scaledHeight}, 0, 0, gocv.InterpolationDefault)
}