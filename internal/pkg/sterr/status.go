package sterr

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	// New returns an error representing c and msg.  If c is OK, returns nil.
	New = status.Error

	// Newf returns New(c, fmt.Sprintf(format, a...)).
	Newf = status.Errorf

	// ErrMissingValue is appended to keyvals slices with odd length to substitute
	// the missing value.
	ErrMissingValue = errors.New("(MISSING)")
)

func WithDetails(e error, keyvals ...interface{}) error {
	if e == nil {
		return nil
	}
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, ErrMissingValue)
	}
	details := map[string]interface{}{}
	for i := 0; i < len(keyvals); i += 2 {
		details[fmt.Sprint(keyvals[i])] = keyvals[i+1]
	}
	st, _ := status.FromError(e)
	return statWithDetails(st, details).Err()
}

func statWithDetails(stat *status.Status, details map[string]interface{}) *status.Status {
	const desc = "invalid details format"

	detailsStruct, err := structpb.NewStruct(details)
	if err != nil {
		newSt, _ := stat.WithDetails(structpb.NewStringValue(fmt.Sprintf("%s: %s", desc, err.Error())))
		return newSt
	}

	statDet, err := stat.WithDetails(detailsStruct)
	if err != nil {
		newSt, _ := stat.WithDetails(structpb.NewStringValue(fmt.Sprintf("%s: %s", desc, err.Error())))
		return newSt
	}
	return statDet
}
