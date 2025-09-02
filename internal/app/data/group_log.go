package data

import (
	"context"
	"fmt"
	"time"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) GetMGroupLogs(ctx context.Context, startDate, endDate time.Time) ([]*models.GroupLog, error) {
	//ctx = boil.WithDebug(ctx, true)
	qmWhere := fmt.Sprintf("%s >= ? AND %s <= ?",
		models.GroupLogTableColumns.CreatedAt,
		models.GroupLogTableColumns.CreatedAt)
	logSlice, err := models.GroupLogs(
		qm.Where(qmWhere, startDate, endDate),
	).All(ctx, s.db)
	if err != nil {
		logger.Errorf(ctx, "error getting mLogs")
		return logSlice, err
	}

	return logSlice, nil
}

// SetMGroupLogs Сохранение
func (s *Store) SetMGroupLogs(ctx context.Context, mLog *models.GroupLog) (*models.GroupLog, error) {
	err := mLog.Insert(ctx, s.db, boil.Blacklist(
		models.GroupLogTableColumns.LogID,
		models.GroupLogTableColumns.DeletedAt))
	if err != nil {
		err = errors.Wrap(err, "Set mLog error insert")
		logger.Warnf(ctx, "SetMAgent: errInsert: '%v'", err)

		return mLog, err
	}

	logger.Infof(ctx, "Set mLog error: '%+v'", mLog)

	return mLog, err
}

func (s *Store) GetLastMGroupLog(ctx context.Context) (returnRunsLog *models.GroupLog, err error) {
	lastLog, err := models.GroupLogs(
		qm.OrderBy(fmt.Sprint(models.GroupLogTableColumns.LogID, " desc")),
	).One(ctx, s.db)
	if err != nil {
		err = errors.Wrapf(err, "Error fetch last log: ")
		logger.Error(ctx, err)
	}

	logger.Debugf(ctx, "GetLastMGroupLog: '%+v'", lastLog)

	return lastLog, err
}
