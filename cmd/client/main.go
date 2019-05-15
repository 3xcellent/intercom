package main

import (
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/3xcellent/intercom/proto"
)

func main() {
	conn, err := grpc.Dial("localhost:6000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := proto.NewIntercomClient(conn)
	stream, err := client.Broadcast(context.Background())
	waitc := make(chan struct{})

	msg := &proto.BroadcastReq{Text: "sup"}

	go func() {
		for {
			log.Println("Sleeping...")
			time.Sleep(2 * time.Second)
			log.Println("Sending msg...")
			stream.Send(msg)
		}
	}()
	<-waitc
	stream.CloseSend()
}
