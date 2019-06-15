# Intercom
Golang GRPC stream server and client for sending video.

This is a demonstration of a bi-directional GRPC stream, while also being somewhat interesting in that it is streaming images via a connected webcam.  I have so far only tested on a Macbook Pro.  While the following installation instructions are for OSX as well,  there is no reason this wouldn't work on other OS's with the proper OpenCV installation and gocv build.

I hope this eventually becomes useful.  It would be great to get this running on a Raspberry Pi with a display, webcam, and microphone attached, but for now, I just want to play with:
* GRPC Streaming
* Threading
* Audio/Video
	
## GRPC
Using protobuf and protoc for code-generation of go files (i.e. `intercom.pb.go`)

To generate updated proto files, run:

```
cd _proto
protoc \
  -I ./ \
  --go_out=plugins=grpc:../proto \
  intercom.proto
```

## Dev Setup
1. Install Opencv
	
	```
	brew install opencv
	```
	
1. Install pk-config if not already installed
	
	```
	brew install pkg-config
	```
	 
1. be sure to have pkg-config var set correctly

	```
	export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig/
	```
	
1. Opencv comes with it's own .pc, move it or simlink it to the PKG_CONFIG_PATH

	```
	ln -s /usr/local/opt/opencv@4/lib/pkgconfig/opencv4.pc $PKG_CONFIG_PATH
	```

1. Add these lines to your environment `.bash_profile` or similiar

	```
	export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig/
	alias opencvflags="pkg-config --cflags --libs opencv"
	```

## Running
1. Start Server
    ```
    cd cmd/server
    go run main.go
    ```
    
1. Start Client
    ```
    cd cmd/client
    go run main.go 0 [path to background image, hopefully a kitten]
    ```
    
    Press [Spacebar] to broadcast
    
    Press [Esc] to exit
