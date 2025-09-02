//go:build tools
// +build tools

package tools

// list packages here to prevent them from removal out of go.mod
import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"

	// proto dependencies
	_ "github.com/googleapis/googleapis/google/example/endpointsapis/goapp"

	// proto tools
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "github.com/kannman/modtools"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"

	_ "github.com/bufbuild/buf/cmd/buf"

	_ "github.com/pressly/goose/v3/cmd/goose"
)
