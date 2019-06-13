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

	outPreviewWidth = screenWidth/4
	outPreviewHeight = screenHeight/4
	outPreviewX = screenHeight - outPreviewHeight - (outPreviewHeight/4)
	outPreviewY = screenWidth - outPreviewWidth - (outPreviewWidth/4)

	inBroadcastWidth = screenWidth/2
	inBroadcastHeight = screenHeight/2
	inBroadcastX = screenHeight/2 - inBroadcastHeight/2 - inBroadcastHeight/4
	inBroadcastY = screenWidth/2 - inBroadcastWidth/2 - inBroadcastWidth/4

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

	window := gocv.NewWindow("Capture Window")
	defer window.Close()

	videoPreviewImg := gocv.NewMatWithSize(outPreviewHeight, outPreviewWidth, gocv.MatTypeCV8UC3)
	defer videoPreviewImg.Close()

	inBroadcastImg := gocv.NewMatWithSize(inBroadcastHeight, inBroadcastWidth, gocv.MatTypeCV8UC3)
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

	isReceivingBroadcast := false
	isBroadcasting := false
	wantToBroadcast := false
	wantToQuit := false

	var webcam *gocv.VideoCapture

	// main program loop
	for {
		displayImg = bgImg.Clone()
		switch  window.WaitKey(1) {
		case 27:
			wantToQuit = true
		case 32:
			wantToBroadcast = !wantToBroadcast
		default:
		}

		if wantToQuit {
			if isBroadcasting {
				isBroadcasting = false
				webcam.Close()
			}
			break
		}

		serverImg := getServerBroadcastImg(serverBroadcastStream)
		if isReceivingBroadcast {
			if serverImg == nil || serverImg.Empty() {
				isReceivingBroadcast = false
				fmt.Println("incoming broadcast ended")
			} else {
				resizeInBroadcastImg(serverImg, &inBroadcastImg)
				updateDisplayImgWithinBroadcast(&displayImg, inBroadcastImg)
			}
		} else {
			if serverImg != nil {
				isReceivingBroadcast = true
				fmt.Println("receiving incoming broadcast")
				fmt.Println("placing window at: %d,%d", inBroadcastX, inBroadcastY)
			}
		}

		if wantToBroadcast {
			if !isBroadcasting {
				webcam, err = gocv.OpenVideoCapture(deviceID)
				if err != nil {
					fmt.Printf("Error opening video capture device: %v\n", deviceID)
					return
				}
				isBroadcasting = true
				fmt.Println("outgoing broadcast starting")
			}
			videoCaptureImg := getVideoCaptureImg(webcam)

			if videoCaptureImg.Empty() {
				if isBroadcasting {
					isBroadcasting = false
					fmt.Println("outgoing broadcast ended")
				}
				continue
			}

			broadcastImg(clientBroadcastStream, videoCaptureImg)
			resizeVideoPreviewImg(videoCaptureImg, &videoPreviewImg)
			updateDisplayWithVideoPreview(&displayImg, videoPreviewImg)
		} else {
			if isBroadcasting {
				isBroadcasting = false
				webcam.Close()
			}
		}

		window.IMShow(displayImg)
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

func updateDisplayImgWithinBroadcast(displayImg *gocv.Mat, mirrorImg gocv.Mat) {
	for x := 0; x < mirrorImg.Size()[0]; x++ {
		for y := 0; y < inBroadcastWidth; y++ {
			displayImg.SetIntAt3(x+inBroadcastX, y+inBroadcastY, 0, mirrorImg.GetIntAt3(x, y, 0))
		}
	}
}

func updateDisplayWithVideoPreview(displayImg *gocv.Mat, mirrorImg gocv.Mat) {
	for x := 0; x < mirrorImg.Size()[0]; x++ {
		for y := 0; y < outPreviewWidth; y++ {
			displayImg.SetIntAt3(x+outPreviewX, y+outPreviewY, 0, mirrorImg.GetIntAt3(x, outPreviewWidth-y, 0))
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
	outPreviewScaledHeight := int(math.Floor(outPreviewWidth/screenCapRatio))
	gocv.Resize(videoCaptureImg, sizedImg, image.Point{X: outPreviewWidth, Y: outPreviewScaledHeight}, 0, 0, gocv.InterpolationDefault)
}


func resizeInBroadcastImg(origImg *gocv.Mat, sizedImg *gocv.Mat) {
	screenCapRatio := float64(float64(origImg.Size()[1])/float64(origImg.Size()[0]))
	scaledHeight := int(math.Floor(inBroadcastWidth/screenCapRatio))
	gocv.Resize(*origImg, sizedImg, image.Point{X: inBroadcastWidth, Y: scaledHeight}, 0, 0, gocv.InterpolationDefault)
}