# Intercom
I hope this becomes useful.

Goals/Things I want to Play With
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
