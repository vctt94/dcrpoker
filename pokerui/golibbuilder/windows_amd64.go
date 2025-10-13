//go:build windows && amd64
// +build windows,amd64

package golibbuilder

//go:generate env CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -buildmode=c-shared -o ../build/windows/amd64/golib.dll ../golib/sharedlib
