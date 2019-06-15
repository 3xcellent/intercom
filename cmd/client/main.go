package main

import (
	"context"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"sync"
	"time"

	"github.com/3xcellent/intercom/proto"
	"gocv.io/x/gocv"
	"google.golang.org/grpc"
)

const (
	screenWidth  = 1280 / 2
	screenHeight = 720 / 2

	outPreviewWidth  = screenWidth / 4
	outPreviewHeight = screenHeight / 4
	outPreviewX      = screenHeight - outPreviewHeight - (outPreviewHeight / 4)
	outPreviewY      = screenWidth - outPreviewWidth - (outPreviewWidth / 4)

	inBroadcastWidth  = screenWidth / 2
	inBroadcastHeight = screenHeight / 2
	inBroadcastX      = screenHeight/2 - inBroadcastHeight/2 - inBroadcastHeight/4
	inBroadcastY      = screenWidth/2 - inBroadcastWidth/2 - inBroadcastWidth/4

	matType = gocv.MatTypeCV8UC3
)

type intercomClient struct {
	window   *gocv.Window
	webcam   *gocv.VideoCapture
	deviceID string

	streamClient proto.Intercom_ConnectClient

	bgImg           gocv.Mat
	displayImg      gocv.Mat
	videoPreviewImg gocv.Mat
	inBroadcastImg  gocv.Mat

	inBroadcastImgSync  sync.Mutex
	videoPreviewImgSync sync.Mutex

	lastInBroadcastTime time.Time

	isReceivingBroadcast bool
	isSendingBroadcast   bool
	wantToBroadcast      bool
	wantToQuit           bool
}

func (c *intercomClient) loadBackgroundImg(path string) {
	c.bgImg = gocv.NewMatWithSize(screenHeight, screenWidth, matType)
	defaultImg := gocv.IMRead(path, gocv.IMReadColor)
	defer defaultImg.Close()

	if defaultImg.Empty() {
		fmt.Printf("Error reading image from: %v\n", path)
		return
	} else {
		fmt.Printf("Opening image from: %v | %#v\n", path, defaultImg.Size())
	}
	gocv.Resize(defaultImg, &c.bgImg, image.Point{X: screenWidth, Y: screenHeight}, 0, 0, gocv.InterpolationDefault)
	c.ResetDisplayImg()
}

func (c *intercomClient) shutdown() {
	if c.isSendingBroadcast {
		c.isSendingBroadcast = false
		c.webcam.Close()
	}

	c.bgImg.Close()
	c.displayImg.Close()
	c.videoPreviewImg.Close()
	c.inBroadcastImg.Close()

	c.window.Close()
}

func (c *intercomClient) connectToServer() {
	// dail server
	conn, err := grpc.Dial(":6000", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	// create streams
	client := proto.NewIntercomClient(conn)
	c.streamClient, err = client.Connect(context.Background())
	if err != nil {
		panic(err)
	}
}

func (c *intercomClient) ResetDisplayImg() {
	c.displayImg = c.bgImg.Clone()
}

func (c *intercomClient) handleKeyEvents() {
	switch c.window.WaitKey(1) {
	case 27:
		c.wantToQuit = true
	case 32:
		c.wantToBroadcast = !c.wantToBroadcast
	default:
	}
}

func (c *intercomClient) handleReceiveBroadcast() {
	for {
		resp, err := c.streamClient.Recv()
		if err == io.EOF {
			c.ResetDisplayImg()
			continue
		}
		if err != nil {
			panic(err)
		}

		c.lastInBroadcastTime = time.Now()

		serverImg, err := gocv.NewMatFromBytes(int(resp.Height), int(resp.Width), gocv.MatType(resp.Type), resp.Bytes)
		if err != nil {
			fmt.Printf("cannot create NewMatFromBytes %v\n", err)
			c.ResetDisplayImg()
			continue
		}
		defer serverImg.Close()

		if serverImg.Empty() {
			c.isReceivingBroadcast = false
			c.ResetDisplayImg()
			fmt.Println("incoming broadcast ended")
			continue
		}

		if !c.isReceivingBroadcast {
			c.isReceivingBroadcast = true
			fmt.Println("receiving incoming broadcast")
		}

		screenCapRatio := float64(float64(serverImg.Size()[1]) / float64(serverImg.Size()[0]))
		scaledHeight := int(math.Floor(inBroadcastWidth / screenCapRatio))

		c.inBroadcastImgSync.Lock()
		gocv.Resize(serverImg, &c.inBroadcastImg, image.Point{X: inBroadcastWidth, Y: scaledHeight}, 0, 0, gocv.InterpolationDefault)
		c.inBroadcastImgSync.Unlock()
	}
}

func (c *intercomClient) handleSendBroadcast() {
	if c.wantToBroadcast {
		c.sendVideoCapture()
	} else {
		if c.isSendingBroadcast {
			c.isSendingBroadcast = false
			c.ResetDisplayImg()
			c.webcam.Close()
		}
	}
}

func (c *intercomClient) sendVideoCapture() {
	if !c.isSendingBroadcast {
		var err error
		c.webcam, err = gocv.OpenVideoCapture(c.deviceID)
		if err != nil {
			fmt.Printf("Error opening video capture device: %v\n", c.deviceID)
			return
		}
		c.isSendingBroadcast = true
		fmt.Println("outgoing broadcast starting")
	}

	videoCaptureImg := gocv.NewMat()
	defer videoCaptureImg.Close()

	if ok := c.webcam.Read(&videoCaptureImg); !ok {
		fmt.Println("didn't read from cam")
	}

	if videoCaptureImg.Empty() {
		if c.isSendingBroadcast {
			c.isSendingBroadcast = false
			fmt.Println("outgoing broadcast ended")
		}
		return
	}

	req := proto.Broadcast{
		Height: int32(videoCaptureImg.Size()[0]),
		Width:  int32(videoCaptureImg.Size()[1]),
		Type:   int32(videoCaptureImg.Type()),
		Bytes:  videoCaptureImg.ToBytes(),
	}

	if err := c.streamClient.Send(&req); err != nil {
		panic(err)
		return
	}

	screenCapRatio := float64(float64(videoCaptureImg.Size()[1]) / float64(videoCaptureImg.Size()[0]))
	outPreviewScaledHeight := int(math.Floor(outPreviewWidth / screenCapRatio))

	c.videoPreviewImgSync.Lock()
	gocv.Resize(videoCaptureImg, &c.videoPreviewImg, image.Point{X: outPreviewWidth, Y: outPreviewScaledHeight}, 0, 0, gocv.InterpolationDefault)
	c.videoPreviewImgSync.Unlock()
}

func (c *intercomClient) draw() {
	if c.hasIncomingBroadcast() {
		c.inBroadcastImgSync.Lock()
		for x := 0; x < c.inBroadcastImg.Size()[0]; x++ {
			for y := 0; y < inBroadcastWidth; y++ {
				c.displayImg.SetIntAt3(x+inBroadcastX, y+inBroadcastY, 0, c.inBroadcastImg.GetIntAt3(x, y, 0))
			}
		}
		c.inBroadcastImgSync.Unlock()
	}

	if c.isSendingBroadcast {
		c.videoPreviewImgSync.Lock()
		for x := 0; x < c.videoPreviewImg.Size()[0]; x++ {
			for y := 0; y < outPreviewWidth; y++ {
				c.displayImg.SetIntAt3(x+outPreviewX, y+outPreviewY, 0, c.videoPreviewImg.GetIntAt3(x, outPreviewWidth-y, 0))
			}
		}
		c.videoPreviewImgSync.Unlock()
	}

	c.window.IMShow(c.displayImg)
}

func (c *intercomClient) hasIncomingBroadcast() bool {
	if !c.isReceivingBroadcast {
		return false
	}
	if time.Now().After(c.lastInBroadcastTime.Add(500 * time.Millisecond)) {
		fmt.Println("incoming broadcast timed out")
		c.isReceivingBroadcast = false
		c.ResetDisplayImg()
		return false
	}
	return true
}

func (c *intercomClient) Run() {
	c.connectToServer()
	go c.handleReceiveBroadcast()

	// main program loop
	for {
		c.handleKeyEvents()

		if c.wantToQuit {
			c.shutdown()
			break
		}

		c.handleSendBroadcast()
		c.draw()
	}
}

func createIntercomClient(vidoeCaptureDeviceId, filename string) intercomClient {
	client := intercomClient{
		window:          gocv.NewWindow("Capture Window"),
		deviceID:        vidoeCaptureDeviceId,
		videoPreviewImg: gocv.NewMatWithSize(outPreviewHeight, outPreviewWidth, gocv.MatTypeCV8UC3),
		inBroadcastImg:  gocv.NewMatWithSize(inBroadcastHeight, inBroadcastWidth, gocv.MatTypeCV8UC3),
	}

	client.loadBackgroundImg(filename)

	return client
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tintercom [camera ID] [path/to/background.img]")
		return
	}
	deviceID := os.Args[1]
	filename := os.Args[2]

	client := createIntercomClient(deviceID, filename)
	client.Run()
}
