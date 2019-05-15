// +build !customenv,!openvino

package gocv

// Changes here should be mirrored in contrib/cgo.go.

/*
#cgo !windows pkg-config: opencv4
#cgo CXXFLAGS:   --std=c++11
#cgo windows  CPPFLAGS:   -IC:/opencv/build/install/include
#cgo windows  LDFLAGS:    -LC:/opencv/build/install/x64/mingw/lib -lopencv_core401 -lopencv_face401 -lopencv_videoio401 -lopencv_imgproc401 -lopencv_highgui401 -lopencv_imgcodecs401 -lopencv_objdetect401 -lopencv_features2d401 -lopencv_video401 -lopencv_dnn401 -lopencv_xfeatures2d401 -lopencv_plot401 -lopencv_tracking401 -lopencv_img_hash401 -lopencv_calib3d401
*/
import "C"
