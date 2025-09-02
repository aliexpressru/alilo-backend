package data

import (
	"context"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func (s *Store) SetMRunReport(
	ctx context.Context,
	mRunReport *models.RunReport,
) (runReportID *models.RunReport, err error) {
	err = mRunReport.Insert(ctx, s.db, boil.Blacklist(
		models.RunReportColumns.DeletedAt,
		models.RunReportColumns.ID,
	))
	if err != nil {
		logger.Warnf(ctx, "SetMRunReport: errInsert: '%+v'", err)

		return nil, err
	}

	return mRunReport, err
}

func (s *Store) GetMRunReport(ctx context.Context, runID int32) (
	mRunReport *models.RunReport, err error) {
	mRunReport, err = models.RunReports(
		models.RunReportWhere.RunID.EQ(runID),
		qm.OrderBy(fmt.Sprint(models.RunReportColumns.UpdatedAt, " desc")),
	).One(ctx, s.db)

	return mRunReport, err
}

func (s *Store) UpdateMRunReport(ctx context.Context, mRunReport *models.RunReport) (
	returnMRunReport *models.RunReport, err error) {
	//logger.Debug(ctx, "RunReport is disabled", debug.Stack())

	if mRunReport != nil {
		_, err = mRunReport.Update(ctx, s.db, boil.Blacklist(
			models.RunReportColumns.CreatedAt,
			models.RunReportColumns.DeletedAt,
		))
		if err != nil {
			logger.Error(ctx, "Errorr: ", err)
		}
	} else {
		logger.Errorf(ctx, "UpdateMRunReport -> mRunReport: '%v'", mRunReport)
	}

	return mRunReport, err
}

func (s *Store) DeleteRunReport(ctx context.Context, mRunReport *models.RunReport) error {
	_, err := mRunReport.Delete(ctx, s.db)
	return err
}
