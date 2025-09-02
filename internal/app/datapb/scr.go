package datapb

import (
	"context"
	errors2 "errors"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) GetPbScript(ctx context.Context, scriptID int32) (*pb.Script, error) {
	mScript, err := s.db.GetMScript(ctx, scriptID)
	if err != nil {
		err = errors.Wrapf(err, "Error fetch mScript")
		logger.Errorf(ctx, err.Error())

		return nil, err
	}

	serverScript, err := conv.ModelToPBScript(ctx, mScript)
	if err != nil {
		err = errors.Wrapf(err, "Error conv mScript to pbScript")
		logger.Warnf(ctx, err.Error())
	}

	return serverScript, err
}

// GetAllEnabledScripts Возвращает массив включенных Scripts
func (s *Store) GetAllEnabledScripts(ctx context.Context, scenarioID int32) (
	returnScripts []*pb.Script, err error) {
	logger.Infof(ctx, "Get all Active PbScripts. Send data, scenarioID: '%+v'", scenarioID)

	mScriptsList, err := s.db.GetAllEnabledMScripts(ctx, scenarioID)
	if err != nil {
		err = errors.Wrapf(err, "Error fetch Active scripts")
		logger.Errorf(ctx, err.Error())

		return returnScripts, err
	}

	for _, mServerScript := range mScriptsList {
		pbScript, er := conv.ModelToPBScript(ctx, mServerScript)
		if er == nil {
			returnScripts = append(returnScripts, pbScript)
		} else {
			er = errors.Wrapf(er, "%v", mServerScript.Name)
			err = errors2.Join(err, er)
			logger.Errorf(ctx, "Active script ModelToPB: '%v' '%v'", pbScript, err.Error())
		}
	}

	logger.Infof(ctx, "Active PbScripts: '%+v'", len(returnScripts))

	return returnScripts, err
}

func (s *Store) GetAllPbScripts(ctx context.Context, scenarioID int32) (returnScripts []*pb.Script, err error) {
	mScripts, err := s.db.GetAllMScripts(ctx, scenarioID)
	if err != nil {
		err = errors.Wrapf(err, "Error fetch All mScripts")
		logger.Errorf(ctx, err.Error())

		return returnScripts, err
	}

	for _, serverScript := range mScripts {
		pbScript, er := conv.ModelToPBScript(ctx, serverScript)
		if er == nil {
			returnScripts = append(returnScripts, pbScript)
		} else {
			er = errors.Wrap(er, serverScript.Name)
			err = errors2.Join(err, er)
			logger.Errorf(ctx, "All script ModelToPB: '%v' '%+v'", pbScript, err)
		}
	}

	return returnScripts, err
}

func (s *Store) CreatePbScript(ctx context.Context, pbScript *pb.Script) (int32, error) {
	mScript, message := conv.PbToModelScript(ctx, pbScript)
	if message != "" {
		return -1, errors.Wrapf(errors.New(message), "Error conversion Script when creating")
	}

	scriptID, err := s.db.CreateMScript(ctx, mScript)

	return scriptID, err
}
