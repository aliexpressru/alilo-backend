package data

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/aarondl/sqlboiler/v4/types"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
)

func (s *Store) CreateMStatisticDump(ctx context.Context) (models.StatisticDump, error) {
	statisticDump := models.StatisticDump{}
	err := statisticDump.Insert(ctx, s.db, boil.Blacklist(
		models.StatisticDumpColumns.StatisticDumpID,
		models.StatisticDumpColumns.CreatedAt,
	))

	return statisticDump, err
}

func (s *Store) DeleteMStatisticDump(ctx context.Context, statisticDumpID int32) error {
	statisticDump, err := models.FindStatisticDump(ctx, s.db, statisticDumpID)
	if err != nil {
		return err
	}

	_, err = statisticDump.Delete(ctx, s.db)
	if err != nil {
		return err
	}

	return err
}

func (s *Store) GetMStatisticDumps(ctx context.Context, from, to *timestamp.Timestamp) (
	[]*models.StatisticDump, error) {
	mod := []qm.QueryMod{
		models.StatisticDumpWhere.CreatedAt.GTE(from.AsTime()),
		models.StatisticDumpWhere.CreatedAt.LTE(to.AsTime()),
		qm.OrderBy(models.StatisticDumpColumns.StatisticDumpID),
	}
	all, err := models.StatisticDumps(mod...).All(ctx, s.db)
	if err != nil {
		return all, err
	}

	return all, nil
}

func (s *Store) GetLustMStatisticDumpID(ctx context.Context) ([]int32, error) {
	mod := []qm.QueryMod{
		qm.OrderBy(fmt.Sprint(models.StatisticDumpColumns.StatisticDumpID, " desc")),
		qm.Limit(2),
	}
	var ids = make([]int32, 0)

	all, err := models.StatisticDumps(mod...).All(ctx, s.db)
	if err != nil {
		return ids, err
	}

	for _, statisticDump := range all {
		ids = append(ids, statisticDump.StatisticDumpID)
	}

	return ids, nil
}

func (s *Store) PutMStatistic(ctx context.Context, statistic *models.Statistic) error {
	err := statistic.Insert(ctx, s.db, boil.Blacklist(
		models.StatisticColumns.StatisticID,
	))

	return err
}

func (s *Store) GetMStatistics(ctx context.Context, statisticDumpIDs []int32) (*models.StatisticSlice, error) {
	logger.Infof(ctx, "Get MStatistics. Come data: '%+v'", statisticDumpIDs)
	mod := []qm.QueryMod{
		models.StatisticWhere.StatisticDumpID.IN(statisticDumpIDs),
		qm.OrderBy(models.StatisticColumns.StatisticID),
		models.StatisticWhere.TraceIds.NEQ(types.StringArray{}),
	}
	statisticSlice, err := models.Statistics(mod...).All(ctx, s.db)
	if err != nil {
		err = errors.Wrapf(err, "Error get MStatistics")
		logger.Warn(ctx, err)

		return nil, err
	}

	return &statisticSlice, err
}

func (s *Store) CountURLsInRun(ctx context.Context, runID int32, scriptRunID int32) (int32, error) {
	var count int32

	ids, err := s.GetLustMStatisticDumpID(ctx)
	if err != nil {
		logger.Warnf(ctx, "error getting dump IDs: %+v", err)
	}
	rawQuery := ""
	if len(ids) == 0 {
		rawQuery = fmt.Sprintf(
			"%s COUNT(*) FROM statistic WHERE (%s && '{%d}'::BIGINT[]) AND (%s && '{%d}'::BIGINT[])",
			"SELECT", models.StatisticColumns.ScriptRunIds, scriptRunID, models.StatisticColumns.RunIds, runID)
	} else {
		var idsS = make([]string, 0)
		for _, value := range ids {
			idsS = append(idsS, strconv.Itoa(int(value)))
		}

		arr := strings.Join(idsS, ", ")
		rawQuery = fmt.Sprintf(
			"%s COUNT(*) FROM statistic WHERE (%s && '{%d}'::BIGINT[]) AND (%s && '{%d}'::BIGINT[]) AND %s in (%s)",
			"SELECT",
			models.StatisticColumns.ScriptRunIds, scriptRunID,
			models.StatisticColumns.RunIds, runID,
			models.StatisticColumns.StatisticDumpID, arr)
	}
	//boil.DebugMode = true
	rows, err := queries.Raw(
		rawQuery).
		QueryContext(ctx, s.db)

	if err != nil {
		return -1, err
	}
	for rows.Next() {
		if err = rows.Scan(&count); err != nil {
			return -1, err
		}
	}

	return count, err
}

func (s *Store) CountURLsInRunNew(ctx context.Context, runID int32, scriptRunID int32) (int32, error) {
	var count int32

	rawQuery := fmt.Sprintf(
		"SELECT cardinality(url_paths) FROM traces WHERE run_script_id = %d AND run_id  = %d ORDER BY id DESC LIMIT 1;",
		scriptRunID,
		runID)
	//boil.DebugMode = true
	rows, err := queries.Raw(
		rawQuery).
		QueryContext(ctx, s.db)

	if err != nil {
		return -1, err
	}
	for rows.Next() {
		if err = rows.Scan(&count); err != nil {
			return -1, err
		}
	}

	return count, err
}
