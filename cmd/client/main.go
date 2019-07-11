package main

import (
	"context"
	"fmt"
	"os"

	"github.com/3xcellent/intercom/cmd/client/intercom"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tintercom [camera ID] [path/to/background.img]")
		return
	}
	deviceID := os.Args[1]
	serverIP := os.Args[2]
	filename := os.Args[3]

	//TODO: handle os shutdown/break in context
	client := intercom.CreateIntercomClient(context.Background(), deviceID, serverIP, filename)
	client.Run()
}
