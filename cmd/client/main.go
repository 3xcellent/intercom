package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/3xcellent/intercom/cmd/client/intercom"
)

func main() {
	runtime.LockOSThread()

	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tintercom [camera ID] [path/to/background.img]")
		return
	}
	deviceID := os.Args[1]
	filename := os.Args[2]

	//TODO: handle os shutdown/break in context
	client := intercom.CreateIntercomClient(context.Background(), deviceID, filename)
	client.Run()
}
