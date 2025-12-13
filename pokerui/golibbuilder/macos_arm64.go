//go:build darwin && arm64
// +build darwin,arm64

package golibbuilder

//go:generate go build -o ../build/macos/arm64/golib.dylib -buildmode=c-shared ../golib/sharedlib
//go:generate cp -r ../build/macos/arm64/golib.dylib ../flutterui/plugin/macos/Frameworks
//go:generate mkdir -p ../flutterui/pokerui/build/macos/Build/Products/Debug/dcrpoker.app/Contents/Frameworks
//go:generate cp -f ../build/macos/arm64/golib.dylib ../flutterui/pokerui/build/macos/Build/Products/Debug/dcrpoker.app/Contents/Frameworks/golib.dylib

//go:generate mkdir -p ../flutterui/pokerui/build/macos/Build/Products/Profile/dcrpoker.app/Contents/Frameworks
//go:generate cp -f ../build/macos/arm64/golib.dylib ../flutterui/pokerui/build/macos/Build/Products/Profile/dcrpoker.app/Contents/Frameworks/golib.dylib
//go:generate mkdir -p ../flutterui/pokerui/build/macos/Build/Products/Release/dcrpoker.app/Contents/Frameworks
//go:generate cp -f ../build/macos/arm64/golib.dylib ../flutterui/pokerui/build/macos/Build/Products/Release/dcrpoker.app/Contents/Frameworks/golib.dylib
