package main

import (
	"fmt"
	"image"
	"os"
	"time"

	"gocv.io/x/gocv"
)

const (
	imgSizeWidth = 1280/2
	imgSizeHeight = 720/2
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
	fmt.Printf("defaultImg Type: %#v\n", defaultImg.Type())
	resizedImage := gocv.NewMatWithSize(imgSizeHeight, imgSizeWidth, gocv.MatTypeCV8UC3)
	fmt.Printf("Created resizedImage: %#v\n", resizedImage.Size())
	gocv.Resize(defaultImg, &resizedImage, image.Point{X: imgSizeWidth, Y: imgSizeHeight}, 0, 0, gocv.InterpolationDefault)
	fmt.Printf("Resized defaultImg: %#v\n", resizedImage.Size())

	if defaultImg.Empty() {
		fmt.Println("Error reading image from: %v\n", filename)
		return
	} else {
		fmt.Println("Opening image from: %v | %#v\n", filename, defaultImg.Size())
	}

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

	switchedAt := time.Now()
	fmt.Printf("Start reading device: %v\n", deviceID)

	IsShowingWaitScreen := true

	for {
		if time.Now().After(switchedAt.Add(5 * time.Second)) {
			switchedAt = time.Now()
			IsShowingWaitScreen = !IsShowingWaitScreen
			fmt.Printf("IsShowingWaitScreen: %v\n", IsShowingWaitScreen)
		}

		if IsShowingWaitScreen {
			showImg = resizedImage.Clone()
		} else {

			screenCap := gocv.NewMat()
			if ok := webcam.Read(&screenCap); !ok {
				fmt.Printf("Device closed: %v\n", deviceID)
				return
			}
			gocv.Resize(screenCap, &screenCap, image.Point{X: imgSizeWidth, Y: imgSizeHeight}, 0, 0, gocv.InterpolationDefault)
			showImg = screenCap.Clone()
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
