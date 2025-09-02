package grpcgw

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

var MarshalerOption = runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
	MarshalOptions: protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
	},
	UnmarshalOptions: protojson.UnmarshalOptions{
		DiscardUnknown: true,
	},
})
