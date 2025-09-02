package datapb

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

// SearchPB Функция поиска в pbScripts и pbSimpleScripts
func (s *Store) SearchPB(ctx context.Context,
	searchQuery string, notLike bool, projectID int32, scenarioID int32, startWith bool, endWith bool) (
	scripts []*pb.Script,
	simpleScripts []*pb.SimpleScript,
	scenarios []*pb.Scenario,
	projects []*pb.Project,
	err error) {
	mScripts, mSimpleScripts, mScenarios, mProjects, err := s.db.SearchM(ctx,
		searchQuery, notLike, projectID, scenarioID, startWith, endWith)
	if err != nil {
		logger.Errorf(ctx, "SearchPB error: %v", err.Error())
		return scripts, simpleScripts, scenarios, projects, err
	}

	if len(mScripts) > 0 {
		for _, mScript := range mScripts {
			pbScript, er := conv.ModelToPBScript(ctx, mScript)
			if er != nil {
				err = errors.Wrapf(er, "ModelToPBScript error: %v", er)
				logger.Errorf(ctx, "SearchPB: %v", err.Error())

				return scripts, simpleScripts, scenarios, projects, err
			}

			scripts = append(scripts, pbScript)
		}
	}

	if len(mSimpleScripts) > 0 {
		for _, mSimpleScript := range mSimpleScripts {
			pbSimpleScript, mes := conv.ModelToPBSimpleScript(ctx, mSimpleScript)
			if mes != "" {
				err = errors.New(mes)
				logger.Errorf(ctx, "ModelToPBSimpleScript error: %v", err.Error())

				return scripts, simpleScripts, scenarios, projects, err
			}

			simpleScripts = append(simpleScripts, pbSimpleScript)
		}
	}

	if len(mScenarios) > 0 {
		for _, mScenario := range mScenarios {
			pbScenario, mes := conv.ModelToPBScenario(ctx, mScenario)
			if mes != "" {
				err = errors.New(mes)
				logger.Errorf(ctx, "ModelToPBScenario error: %v", err.Error())

				return scripts, simpleScripts, scenarios, projects, err
			}

			scenarios = append(scenarios, pbScenario)
		}
	}

	if len(mProjects) > 0 {
		for _, mProject := range mProjects {
			pbProject, mes := conv.ModelToPBProject(ctx, mProject)
			if mes != "" {
				err = errors.New(mes)
				logger.Errorf(ctx, "ModelToPBProject error: %v", err.Error())

				return scripts, simpleScripts, scenarios, projects, err
			}

			projects = append(projects, pbProject)
		}
	}

	logger.Infof(ctx, "Found: { scripts:%v; simpleScripts:%v; scenarios:%v; projects:%v }",
		len(scripts), len(simpleScripts), len(scenarios), len(projects))

	return scripts, simpleScripts, scenarios, projects, nil
}
