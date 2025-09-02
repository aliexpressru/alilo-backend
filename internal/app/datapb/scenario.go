package datapb

import (
	"context"
	"fmt"

	"github.com/aarondl/null/v8"
	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func (s *Store) GetAllScenarios(ctx context.Context, projectID int32, limit int32, pageNumber int32) (
	scenarios []*pb.Scenario, pages int64, message string) {
	logger.Infof(ctx, "Get all scenarios. Send data: '%+v'", projectID)
	offset, limit := data.OffsetCalculation(limit, pageNumber)
	logger.Infof(ctx, "Query params(Scenarios): (limit:'%v'; offset:'%v')", limit, offset)

	mScenariosList, totalPages, err := s.db.GetMScenariosPaging(ctx, projectID, limit, offset)
	if err != nil {
		message = fmt.Sprintf("Error getting all scenarios: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return scenarios, totalPages, message
	}

	pbScenarios := make([]*pb.Scenario, 0, len(mScenariosList))

	for _, serverScenario := range mScenariosList {
		pbScenario, mes := conv.ModelToPBScenario(ctx, serverScenario)
		message = fmt.Sprint(message, mes)

		if mes == "" {
			pbScenarios = append(pbScenarios, pbScenario)
		} else {
			logger.Errorf(ctx, "ModelToPBScenario: '%v' '%v'", pbScenario, mes)
		}
		pbScenario.LastRunStatus, _, _ = s.GetScenarioLastRunStatus(ctx, pbScenario.ScenarioId)
	}

	logger.Infof(ctx, "len(mScenarios): '%v'", len(mScenariosList))
	logger.Infof(ctx, "len(pbScenarios): '%v'", len(pbScenarios))
	logger.Infof(ctx, "Total Pages Scenarios: '%v'", totalPages)

	return pbScenarios, totalPages, message
}

func (s *Store) GetScenario(ctx context.Context, scenarioID int32) (scenario *pb.Scenario, message string) {
	logger.Infof(ctx, "Get scenario. Come data: '%v'", scenarioID)

	serverScenario, err := s.db.GetMScenario(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Error get mScenario'%v': '%+v'", scenarioID, err.Error())
		logger.Warnf(ctx, "GetScenario %v", message)

		return nil, message
	}

	pbScenario, message := conv.ModelToPBScenario(ctx, serverScenario)
	if message != "" {
		logger.Errorf(ctx, "Get scenario. Prepared model: '%+v' '%v'", pbScenario, message)
	}
	pbScenario.LastRunStatus, _, _ = s.GetScenarioLastRunStatus(ctx, pbScenario.ScenarioId)

	return pbScenario, message
}

func (s *Store) GetScenarioLastRunStatus(ctx context.Context, scenarioID int32) (
	lastRunStatus string, lastRunID int32, message string) {
	logger.Infof(ctx, "Get Scenario Last Run Status. Send data: '%+v'", scenarioID)

	mRun, err := s.db.GetLastMRunning(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Get Scenario Last Run Status: '%v'", err.Error())
		logger.Warnf(ctx, message)

		return lastRunStatus, lastRunID, message
	}
	if mRun == nil {
		return "", 0, ""
	}

	return mRun.Status, mRun.RunID, message
}

func (s *Store) GetScenarioTitle(ctx context.Context, scenarioID int32) (title string) {
	pbScenario, mess := s.GetScenario(ctx, scenarioID)
	logger.Infof(ctx, "GetScenarioTitle -> GetScenario:{mess:'%v', pbScenario:'%+v'", mess, pbScenario)

	if mess == "" &&
		pbScenario.Title != "" { // если нет ошибки при получении сценария, и есть заголовок у полученного сценария
		title = pbScenario.Title
	} else {
		title = "Alilo scenario"
	}

	return title
}

func (s *Store) UpdateScenario(ctx context.Context, pbScenario *pb.Scenario) (err error) {
	mScenario, err := s.db.GetMScenario(ctx, pbScenario.GetScenarioId())
	if err != nil {
		logger.Warnf(ctx, "UpdateScenario -> PbToModelsScenario fail: '%v'", err.Error())

		return err
	}
	mScenario.Title = pbScenario.Title
	mScenario.Descrip = null.StringFrom(pbScenario.Descrip)

	err = s.db.UpdateMScenario(ctx, mScenario)
	if err != nil {
		logger.Warnf(ctx, "UpdateScenario -> UpdateMScenario fail: '%v'", err.Error())

		return err
	}

	return err
}
