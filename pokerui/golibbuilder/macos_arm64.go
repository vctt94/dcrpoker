//go:build darwin && arm64
// +build darwin,arm64

package golibbuilder

//go:generate go build -o ../build/macos/arm64/golib.dylib -buildmode=c-shared ../golib/sharedlib
//go:generate mkdir -p ../flutterui/plugin/macos/Frameworks
//go:generate cp -f ../build/macos/arm64/golib.dylib ../flutterui/plugin/macos/Frameworks/golib.dylib
