package intercom

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/3xcellent/intercom/proto"
	"github.com/gordonklaus/portaudio"
	"github.com/lazywei/go-opencv/opencv"
	"google.golang.org/grpc"
)

const (
	// MAIN WINDOW DIMENSIONS
	screenWidth  = 1280 / 2
	screenHeight = 720 / 2

	// OUTGOING BROADCAST IMAGE (mirror)
	outPreviewWidth  = screenWidth / 4
	outPreviewHeight = screenHeight / 4
	outPreviewX      = screenWidth - outPreviewWidth - (outPreviewWidth / 4)
	outPreviewY      = screenHeight - outPreviewHeight - (outPreviewHeight / 4)

	// INCOMING BROADCAST IMAGE
	inBroadcastWidth  = screenWidth / 2
	inBroadcastHeight = screenHeight / 2
	inBroadcastX      = screenWidth/2 - inBroadcastWidth/2 - inBroadcastWidth/4
	inBroadcastY      = screenHeight/2 - inBroadcastHeight/2 - inBroadcastHeight/4

	// MAT IMAGE TYPE
	//matType = gocv.MatTypeCV8UC3

	// AUDIO SETTINGS
	sampleRate     = 44100
	buffer_seconds = 1
)

type intercomClient struct {
	window *opencv.Window
	webcam *opencv.Capture
	//audioBuffer []float32
	//audioStream *portaudio.Stream
	deviceID  int
	bgImgPath string

	context context.Context

	streamClient proto.Intercom_ConnectClient

	bgImg           *opencv.IplImage
	displayImg      *opencv.IplImage
	videoPreviewImg *opencv.IplImage
	inBroadcastImg  *opencv.IplImage

	lastInBroadcastTime time.Time

	isReceivingBroadcast bool
	isSendingBroadcast   bool
	wantToBroadcast      bool
	wantToQuit           bool
}

func (c *intercomClient) loadBackgroundImg(path string) {
	image := opencv.LoadImage(path)
	if image == nil {
		panic("LoadImage fail")
	}
	c.bgImg = opencv.Resize(image, screenWidth, screenHeight, 1)
	c.ResetDisplayImg()
}

func (c *intercomClient) connectToServer() {
	// dail server
	conn, err := grpc.Dial(":6000", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	// create streams
	client := proto.NewIntercomClient(conn)
	c.streamClient, err = client.Connect(c.context)
	if err != nil {
		panic(err)
	}
}

func (c *intercomClient) ResetDisplayImg() {
	fmt.Print("resetting displayImg")
	c.displayImg = c.bgImg.Clone()

}

func (c *intercomClient) handleKeyEvents() {
	//exit if context is done
	//or continue
	switch opencv.WaitKey(10) {
	case 27:
		c.wantToQuit = true
	case 32:
		c.wantToBroadcast = !c.wantToBroadcast
	default:
	}
}

func (c *intercomClient) handleReceiveBroadcast() {
	//for {
	//	resp, err := c.streamClient.Recv()
	//	if err == io.EOF {
	//		c.ResetDisplayImg()
	//		continue
	//	}
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	c.lastInBroadcastTime = time.Now()
	//
	//	serverImg, err := gocv.NewMatFromBytes(int(resp.Height), int(resp.Width), gocv.MatType(resp.Type), resp.Bytes)
	//	if err != nil {
	//		fmt.Printf("cannot create NewMatFromBytes %v\n", err)
	//		c.ResetDisplayImg()
	//		continue
	//	}
	//	defer serverImg.Releast()
	//
	//	if serverImg.Empty() {
	//		c.isReceivingBroadcast = false
	//		c.ResetDisplayImg()
	//		fmt.Println("incoming broadcast ended")
	//		continue
	//	}
	//
	//	if !c.isReceivingBroadcast {
	//		c.isReceivingBroadcast = true
	//		fmt.Println("receiving incoming broadcast")
	//	}
	//
	//	screenCapRatio := float64(float64(serverImg.Size()[1]) / float64(serverImg.Size()[0]))
	//	scaledHeight := int(math.Floor(inBroadcastWidth / screenCapRatio))
	//
	//	gocv.Resize(serverImg, &c.inBroadcastImg, image.Point{X: inBroadcastWidth, Y: scaledHeight}, 0, 0, gocv.InterpolationDefault)
	//}
}

func (c *intercomClient) handleSendBroadcast() {
	if c.wantToBroadcast {
		c.sendVideoCapture()
		//c.sendAudioBuffer()
	} else {
		if c.isSendingBroadcast {
			c.isSendingBroadcast = false
			c.ResetDisplayImg()
			c.webcam.Release()
			//c.audioStream.Close()
		}
	}
}

//func (c *intercomClient) sendAudioBuffer() {
//	if !c.isReceivingBroadcast {
//		buffer := make([]float32, sampleRate*buffer_seconds)
//		audioStreamFunc := func(in []float32) {
//			for i := range buffer {
//				buffer[i] = in[i]
//			}
//			fmt.Print(".")
//		}
//		var err error
//		fmt.Println("opening stream")
//		c.audioStream, err = portaudio.OpenDefaultStream(1, 0, sampleRate, len(buffer), audioStreamFunc)
//		if err != nil {
//			panic("error recording audio: " + err.Error())
//		}
//		c.audioStream.Start()
//	}
//
//	//TODO send
//	//TODO update proto to accept audio also?
//
//}

func (c *intercomClient) sendVideoCapture() {
	if !c.isSendingBroadcast {
		c.webcam = opencv.NewCameraCapture(c.deviceID)
		if c.webcam == nil {
			return
		}
		c.isSendingBroadcast = true
		fmt.Println("outgoing broadcast starting")
	}

	var img *opencv.IplImage

	if c.webcam.GrabFrame() {
		img = c.webcam.RetrieveFrame(1)
		if img == nil {
			c.isSendingBroadcast = false
			fmt.Println("outgoing broadcast ended")
			return
		}
	}

	req := proto.Broadcast{
		Height: int32(img.Height()),
		Width:  int32(img.Height()),
		Type:   int32(opencv.IPL_DEPTH_8U),
		Bytes:  nil,
	}

	if err := c.streamClient.Send(&req); err != nil {
		panic(err)
		return
	}

	screenCapRatio := float64(float64(img.Width()) / float64(img.Height()))
	outPreviewScaledHeight := int(math.Floor(outPreviewWidth / screenCapRatio))

	c.videoPreviewImg = opencv.Resize(img, outPreviewWidth, outPreviewScaledHeight, 1)
}

func (c *intercomClient) draw() {
	//if c.hasIncomingBroadcast() {
	//	for x := 0; x < c.inBroadcastImg.Size()[0]; x++ {
	//		for y := 0; y < inBroadcastWidth; y++ {
	//			c.displayImg.SetIntAt3(x+inBroadcastX, y+inBroadcastY, 0, c.inBroadcastImg.GetIntAt3(x, y, 0))
	//		}
	//	}
	//}
	//
	if c.isSendingBroadcast && c.videoPreviewImg != nil {
		for x := 1; x < c.videoPreviewImg.Width(); x++ {
			for y := 1; y < c.videoPreviewImg.Height() ; y++ {
				//c.displayImg.SetIntAt3(x+outPreviewX, y+outPreviewY, 0, c.videoPreviewImg.GetIntAt3(x, outPreviewWidth-y, 0))
				//val := c.videoPreviewImg.Get3D(x, outPreviewWidth-y, 0)
				val := c.videoPreviewImg.Get2D(outPreviewWidth-x, y)
				c.displayImg.Set2D(x+outPreviewX, y+outPreviewY, val)
			}
		}
	}

	c.window.ShowImage(c.displayImg)
}

//
func (c *intercomClient) hasIncomingBroadcast() bool {
	if !c.isReceivingBroadcast {
		return false
	}
	if time.Now().After(c.lastInBroadcastTime.Add(300 * time.Millisecond)) {
		c.isReceivingBroadcast = false
		c.ResetDisplayImg()
		return false
	}
	return true
}

func (c *intercomClient) shutdown() {
	if c.isSendingBroadcast {
		c.isSendingBroadcast = false
		c.webcam.Release()
		//c.audioStream.Close()
	}

	c.bgImg.Release()
	c.displayImg.Release()
	c.videoPreviewImg.Release()
	c.inBroadcastImg.Release()

	c.bgImg.Release()
	c.window.Destroy()
}

func (c *intercomClient) Run() {
	c.window = opencv.NewWindow("Intercom")
	c.loadBackgroundImg(c.bgImgPath)
	c.connectToServer()
	go c.handleReceiveBroadcast()

	fmt.Println("Initializing portaudio")
	portaudio.Initialize()
	defer portaudio.Terminate()

	// main program loop
	for {
		select {
		case <-c.context.Done():
			c.wantToQuit = true
			return
		default:
		}

		c.handleKeyEvents()

		if c.wantToQuit {
			c.shutdown()
			break
		}

		c.handleSendBroadcast()
		c.draw()
	}
}

func CreateIntercomClient(ctx context.Context, vidoeCaptureDeviceId int, filename string) intercomClient {
	client := intercomClient{
		deviceID:  vidoeCaptureDeviceId,
		bgImgPath: filename,
		context:   ctx,
	}

	return client
}
