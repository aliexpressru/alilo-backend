package data

import (
	"context"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func (s *Store) GetMTraces(ctx context.Context, from, to *timestamp.Timestamp) (
	[]*models.Trace, error) {
	mod := []qm.QueryMod{
		models.TraceWhere.TraceTime.GTE(from.AsTime()),
		models.TraceWhere.TraceTime.LTE(to.AsTime()),
	}
	all, err := models.Traces(mod...).All(ctx, s.db)
	if err != nil {
		return all, err
	}

	return all, nil
}

func (s *Store) GetMTracesByScriptID(ctx context.Context, scriptID int32) (
	[]*models.Trace, error) {
	mod := []qm.QueryMod{
		models.TraceWhere.ScriptID.EQ(scriptID),
		qm.OrderBy(fmt.Sprint(models.TraceColumns.TraceTime, " desc")),
	}
	all, err := models.Traces(mod...).All(ctx, s.db)
	if err != nil {
		return all, err
	}

	return all, nil
}

func (s *Store) GetMTracesByScriptRunID(ctx context.Context, scriptRunID int32) (
	[]*models.Trace, error) {
	mod := []qm.QueryMod{
		models.TraceWhere.RunScriptID.EQ(scriptRunID),
		qm.OrderBy(fmt.Sprint(models.TraceColumns.TraceTime, " desc")),
	}
	all, err := models.Traces(mod...).All(ctx, s.db)
	if err != nil {
		return all, err
	}

	return all, nil
}

func (s *Store) GetMTracesByTraceID(ctx context.Context, traceID string) (
	[]*models.Trace, error) {
	mod := []qm.QueryMod{
		models.TraceWhere.TraceID.EQ(traceID),
		qm.OrderBy(fmt.Sprint(models.TraceColumns.TraceTime, " desc")),
	}
	all, err := models.Traces(mod...).All(ctx, s.db)
	if err != nil {
		return all, err
	}

	return all, nil
}

// PutMTraces запись трейсов в БД
func (s *Store) PutMTraces(ctx context.Context, traces []*models.Trace) (err error) {
	for _, trace := range traces {
		var count int64
		count, err = models.Traces(models.TraceWhere.TraceID.EQ(trace.TraceID)).Count(ctx, s.db)
		if err != nil {
			logger.Errorf(ctx, "Error check count trace{%+v}", trace.TraceID)

			continue
		}

		// не записываем уже имеющийся трейс в БД
		if count > 0 {
			logger.Debugf(ctx, "Miss insert trace{%+v}, already exist", trace)

			continue
		}

		err = trace.Insert(ctx, s.db, boil.Blacklist(
			models.TraceTableColumns.ID,
			models.TraceColumns.TraceTime,
		))
		if err != nil {
			logger.Errorf(ctx, "Error insert trace{%+v}", trace)
		}
	}

	return err
}
