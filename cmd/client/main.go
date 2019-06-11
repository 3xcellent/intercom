package main

import (
	"fmt"
	"image"
	"math"
	"os"

	"gocv.io/x/gocv"
)

const (
	screenWidth  = 1280/2
	screenHeight = 720/2
	mirrorWindowWidth = screenWidth/4
	mirrorWindowHeight = screenHeight/4
	mirrorWindowX = screenHeight - mirrorWindowHeight - (mirrorWindowHeight/4)
	mirrorWindowY = screenWidth - mirrorWindowWidth - (mirrorWindowWidth/4)
)


func main() {
	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tintercom [camera ID] [path/to/background_img]")
		return
	}
	deviceID := os.Args[1]
	filename := os.Args[2]

	// prepare displayImg
	bgImg := gocv.NewMatWithSize(screenHeight, screenWidth, gocv.MatTypeCV8UC3)
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

	// main loop
	for {
		updateVideoPreviewImg(webcam, &videoPreviewImg)
		updateDisplayWithVideoPreview(&displayImg, videoPreviewImg)

		if displayImg.Empty() {
			fmt.Printf("no image; continue... \n")
			continue
		}

		window.IMShow(displayImg)
		if window.WaitKey(1) == 27 {
			break
		}
	}
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

func updateDisplayWithVideoPreview(displayImg *gocv.Mat, mirrorImg gocv.Mat) {
	for x := 0; x <= mirrorImg.Size()[0]-1; x++ {
		for y := 0; y <= mirrorWindowWidth; y++ {
			displayImg.SetIntAt3(x+mirrorWindowX, y+mirrorWindowY, 0, mirrorImg.GetIntAt3(x, mirrorWindowWidth-y, 0))
		}
	}
}

func updateVideoPreviewImg(webcam *gocv.VideoCapture, sizedImg *gocv.Mat) {
	videoCapture := gocv.NewMat()
	defer videoCapture.Close()

	if ok := webcam.Read(&videoCapture); !ok {
		return
	}
	screenCapRatio := float64(float64(videoCapture.Size()[1])/float64(videoCapture.Size()[0]))
	mirrorWindowScaledHeight := int(math.Floor(mirrorWindowWidth/screenCapRatio))
	gocv.Resize(videoCapture, sizedImg, image.Point{X: mirrorWindowWidth, Y: mirrorWindowScaledHeight}, 0, 0, gocv.InterpolationDefault)
}
