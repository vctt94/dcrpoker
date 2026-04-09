//go:build windows && amd64
// +build windows,amd64

package golibbuilder

//go:generate go build -buildmode=c-shared -o ../build/windows/amd64/golib.dll ../golib/sharedlib
//go:generate powershell -NoProfile -Command "New-Item -ItemType Directory -Force ../flutterui/plugin/windows/libs | Out-Null; Copy-Item -Force ../build/windows/amd64/golib.dll ../flutterui/plugin/windows/libs/golib.dll"
