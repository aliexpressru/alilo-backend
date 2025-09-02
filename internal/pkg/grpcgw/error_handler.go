package grpcgw

import (
	"context"
	"io"
	"net/http"

	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var HTTPProtoErrorHandlerOption = runtime.WithErrorHandler(HTTPProtoErrorHandler)

// HTTPProtoErrorHandler is an implementation of HTTPError.
// If "err" is an error from gRPC system, the function replies with the status code mapped by HTTPStatusFromCode.
// If otherwise, it replies with http.StatusInternalServerError.
//
// The response body returned by this function is a ErrorResponse message marshaled by a Marshaler.
func HTTPProtoErrorHandler(
	ctx context.Context,
	_ *runtime.ServeMux,
	marshaler runtime.Marshaler,
	w http.ResponseWriter,
	_ *http.Request,
	e error,
) {
	//logErrToSpan(ctx, e)

	s, ok := status.FromError(e)
	if !ok {
		s = status.New(codes.Unknown, e.Error())
	}

	w.Header().Del("Trailer")
	w.Header().Set("Content-type", marshaler.ContentType(struct{}{}))
	// handle error details
	detailsStruct, err := getErrDetails(s, marshaler)
	if err != nil {
		fallbackErrorHandler(w, s)
		return
	}

	errorResponse := &pb.ErrorResponse{
		Error: &pb.Error{
			Code:    int64(s.Code()),
			Message: s.Message(),
			Details: detailsStruct,
		},
	}

	buf, err := marshaler.Marshal(errorResponse)
	if err != nil {
		grpclog.Infof("Failed to marshal error message %q: %v", errorResponse, err)
		fallbackErrorHandler(w, s)
		return
	}

	st := runtime.HTTPStatusFromCode(s.Code())
	w.WriteHeader(st)
	if _, err = w.Write(buf); err != nil {
		grpclog.Infof("Failed to write response: %v", err)
	}
}

func getErrDetails(s *status.Status, marshaler runtime.Marshaler) (*structpb.Struct, error) {
	var detailsStruct *structpb.Struct
	if len(s.Proto().Details) == 0 {
		return detailsStruct, nil
	}

	var statDetails = s.Details()

	if len(statDetails) == 1 {
		if pbStruct, ok := statDetails[0].(*structpb.Struct); ok {
			return pbStruct, nil
		}
	}

	// convert statDetails to *structpb.Struct
	// wrap details list to map according to SOA Convention
	var dataDetails interface{} = map[string]interface{}{
		"data": statDetails,
	}
	details, err := marshaler.Marshal(dataDetails)
	if err != nil {
		grpclog.Infof("Failed to marshal details of error message %q: %v", dataDetails, err)
		return nil, err
	}
	if err = marshaler.Unmarshal(details, &detailsStruct); err != nil {
		grpclog.Infof("Failed to unmarshal details of error message %q: %v", details, err)
		return nil, err
	}
	return detailsStruct, nil
}

func fallbackErrorHandler(w http.ResponseWriter, _ *status.Status) {
	const fallback = `{"data": null, "error": {"code": 13, "message": "failed to marshal error message"}}`

	w.WriteHeader(http.StatusInternalServerError)
	if _, err := io.WriteString(w, fallback); err != nil {
		grpclog.Infof("Failed to write response: %v", err)
	}
}

// todo open source  refactoring ???
//func logErrToSpan(ctx context.Context, e error) {
//	if sp := trace.SpanFromContext(ctx); sp != nil && sp.SpanContext().IsValid() {
//		sp.AddEvent("http.error", trace.WithAttributes(attribute.String("http.error", fmt.Sprint(e))))
//	}
//}
