//go:build windows && amd64
// +build windows,amd64

package golibbuilder

// Build the DLL
//go:generate go build -buildmode=c-shared -o ../build/windows/amd64/golib.dll ../golib/sharedlib

//  Copy into the built runner output (Release) so your .exe finds it next to the exe
//    This matches: build\windows\x64\runner\Release\dcrpoker.exe
//go:generate powershell -NoProfile -Command "New-Item -ItemType Directory -Force ../flutterui/pokerui/build/windows/x64/runner/Release | Out-Null; Copy-Item -Force ../build/windows/amd64/golib.dll ../flutterui/pokerui/build/windows/x64/runner/Release/golib.dll"

// Optional: also copy to Debug/Profile if you build those
//go:generate powershell -NoProfile -Command "New-Item -ItemType Directory -Force ../flutterui/pokerui/build/windows/x64/runner/Debug | Out-Null; Copy-Item -Force ../build/windows/amd64/golib.dll ../flutterui/pokerui/build/windows/x64/runner/Debug/golib.dll"
//go:generate powershell -NoProfile -Command "New-Item -ItemType Directory -Force ../flutterui/pokerui/build/windows/x64/runner/Profile | Out-Null; Copy-Item -Force ../build/windows/amd64/golib.dll ../flutterui/pokerui/build/windows/x64/runner/Profile/golib.dll"
