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
	fmt.Printf("Starting with args: %#v\n", os.Args)
	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tcapwindow [camera ID]")
		return
	}

	// parse args
	deviceID := os.Args[1]
	filename := os.Args[2]

	defaultImg := gocv.IMRead(filename, gocv.IMReadColor)

	if defaultImg.Empty() {
		fmt.Println("Error reading image from: %v\n", filename)
		return
	} else {
		fmt.Println("Opening image from: %v | %#v\n", filename, defaultImg.Size())
		fmt.Printf("defaultImg Type: %#v\n", defaultImg.Type())
	}
	resizedImage := gocv.NewMatWithSize(screenHeight, screenWidth, gocv.MatTypeCV8UC3)
	gocv.Resize(defaultImg, &resizedImage, image.Point{X: screenWidth, Y: screenHeight}, 0, 0, gocv.InterpolationDefault)
	fmt.Printf("Resized defaultImg: %#v\n", resizedImage.Size())

	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	window := gocv.NewWindow("Capture Window")
	fmt.Printf("Created Window\n")
	defer window.Close()

	showImg := gocv.NewMat()
	defer showImg.Close()

	fmt.Printf("Start reading device: %v\n", deviceID)
	for {
		showImg = resizedImage.Clone()

		screenCap := gocv.NewMat()
		windowImg := gocv.NewMatWithSize(mirrorWindowHeight, mirrorWindowWidth, gocv.MatTypeCV8UC3)

		if ok := webcam.Read(&screenCap); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		screenCapRatio := float64(float64(screenCap.Size()[1])/float64(screenCap.Size()[0]))
		mirrorWindowScaledHeight := int(math.Floor(mirrorWindowWidth/screenCapRatio))
		gocv.Resize(screenCap, &windowImg, image.Point{X: mirrorWindowWidth, Y: mirrorWindowScaledHeight}, 0, 0, gocv.InterpolationDefault)

		for x := 0; x <= mirrorWindowScaledHeight-1; x++ {
			for y := 0; y <= mirrorWindowWidth; y++ {
				showImg.SetIntAt3(x+mirrorWindowX, y+mirrorWindowY, 0, windowImg.GetIntAt3(x, y, 0))
			}
		}

		if showImg.Empty() {
			fmt.Printf("no image; continue... \n")
			continue
		}

		window.IMShow(showImg)
		if window.WaitKey(1) == 27 {
			break
		}
	}
}
