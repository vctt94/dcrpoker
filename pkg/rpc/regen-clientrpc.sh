#!/bin/sh

set -e

# Temporary binary dir
BINDIR=$(mktemp -d)

export GOBIN=$BINDIR

# --- Build required protoc plugins ---

echo "Building protoc plugins into $BINDIR ..."

# Modern Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Dart plugin (make sure Dart SDK is installed)
dart pub global activate protoc_plugin

# Add both Go and Dart plugin paths to PATH
export PATH="$BINDIR:$HOME/.pub-cache/bin:$PATH"

# --- Generate code ---

echo "Generating Go code..."
protoc --go_out=. --go-grpc_out=. -I. poker.proto

echo "Generating Dart code..."
protoc --dart_out=grpc:../../pokerui/flutterui/plugin/lib/grpc/generated -I. poker.proto

echo "✅ Done. Generated Go and Dart gRPC bindings."
