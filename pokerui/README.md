# pongui - Pong UI (graphical)

# Building

Requires [flutter](https://wwww.flutter.dev). Requires xcode for macos/ios.
Requires Android NDK to build for android. 

## Native Desktop

This will build the desktop version for the current system (no cross-compiling
yet).

Replace `linux` with either `macos` or `windows`.

```shell
$ go generate ./golibbuilder
$ cd flutterui/pongui
$ flutter build linux
```
