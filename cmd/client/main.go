package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/3xcellent/intercom/cmd/client/intercom"
)

func main() {
	runtime.LockOSThread()

	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tintercom [camera ID] [path/to/background.img]")
		return
	}

	deviceId, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		panic("camera device id is not valid: " + err.Error())
	}

	filename := os.Args[2]

	//TODO: handle os shutdown/break in context
	client := intercom.CreateIntercomClient(context.Background(), int(deviceId), filename)
	client.Run()
}
