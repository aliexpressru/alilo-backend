package datapb

import (
	"context"
	"time"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func (s *Store) GetAllGroupLog(ctx context.Context, startDate, endDate time.Time) (
	groupLogs []*pb.GroupLog, err error) {
	logs, err := s.db.GetMGroupLogs(ctx, startDate, endDate)
	if err != nil {
		logger.Errorf(ctx, "error GetMGroupLogs", err)
		return groupLogs, err
	}
	for _, log := range logs {
		pbLog, er := conv.ModelGroupLogToPB(ctx, log)
		if er != nil {
			logger.Errorf(ctx, "", err)

			return nil, err
		}
		groupLogs = append(groupLogs, pbLog)
	}

	return groupLogs, err
}

func (s *Store) GetLastGroupLog(ctx context.Context) (groupLog *pb.GroupLog, err error) {
	mGroupLog, err := s.db.GetLastMGroupLog(ctx)
	if err != nil {
		logger.Errorf(ctx, "error GetLastMGroupLog", err)

		return groupLog, err
	}
	pbLog, er := conv.ModelGroupLogToPB(ctx, mGroupLog)
	if er != nil {
		logger.Errorf(ctx, "error ModelGroupLogToPB", err)

		return pbLog, err
	}

	return pbLog, err
}

func (s *Store) SetGroupLogs(ctx context.Context, log *pb.GroupLog) (groupLog *pb.GroupLog, err error) {
	pbGroupLog, err := conv.PBGroupLogToModel(ctx, log)
	if err != nil {
		logger.Errorf(ctx, "error PBGroupLogToModel", err)

		return groupLog, err
	}
	mGroupLog, err := s.db.SetMGroupLogs(ctx, pbGroupLog)
	if err != nil {
		logger.Errorf(ctx, "error SetMGroupLogs", err)

		return groupLog, err
	}
	groupLog, err = conv.ModelGroupLogToPB(ctx, mGroupLog)
	if err != nil {
		logger.Errorf(ctx, "error ModelGroupLogToPB", err)

		return groupLog, err
	}

	return groupLog, err
}
