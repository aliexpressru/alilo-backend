package datapb

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	v1 "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func (s *Store) GetPBTraces(ctx context.Context, from, to *timestamp.Timestamp) ([]*v1.Trace, error) {
	mTraces, err := s.db.GetMTraces(ctx, from, to)
	if err != nil {
		return []*v1.Trace{}, err
	}

	return conv.ModelsToPbTraces(ctx, mTraces)
}

func (s *Store) GetPBTracesByScriptID(ctx context.Context, scriptID int32) ([]*v1.Trace, error) {
	mTraces, err := s.db.GetMTracesByScriptID(ctx, scriptID)
	if err != nil {
		return []*v1.Trace{}, err
	}

	return conv.ModelsToPbTraces(ctx, mTraces)
}

func (s *Store) GetPBTracesByScriptRunID(ctx context.Context, scriptRunID int32) ([]*v1.Trace, error) {
	mTraces, err := s.db.GetMTracesByScriptRunID(ctx, scriptRunID)
	if err != nil {
		return []*v1.Trace{}, err
	}

	return conv.ModelsToPbTraces(ctx, mTraces)
}

func (s *Store) GetPBTracesByTraceID(ctx context.Context, traceID string) ([]*v1.Trace, error) {
	mTraces, err := s.db.GetMTracesByTraceID(ctx, traceID)
	if err != nil {
		return []*v1.Trace{}, err
	}

	return conv.ModelsToPbTraces(ctx, mTraces)
}
