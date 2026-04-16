//go:build linux && amd64
// +build linux,amd64

package golibbuilder

//go:generate go build -o ../build/linux/amd64/golib.so -buildmode=c-shared ../golib/sharedlib
//go:generate mkdir -p ../flutterui/plugin/linux/libs
//go:generate cp -f ../build/linux/amd64/golib.so ../flutterui/plugin/linux/libs/golib.so
