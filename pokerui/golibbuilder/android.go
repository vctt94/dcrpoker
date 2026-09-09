//go:build android
// +build android

package golibbuilder

//go:generate mkdir -p ../build/android
//go:generate gomobile bind -target android -androidapi 21 -o ../build/android/golib.aar ../golib
//go:generate mkdir -p ../flutterui/plugin/android/golib/libs
//go:generate cp -f ../build/android/golib.aar ../flutterui/plugin/android/golib/libs/golib.aar
