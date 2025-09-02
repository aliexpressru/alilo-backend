package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/aliexpressru/alilo-backend/internal/app/config"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/pkg/errors"
)

//	createMCommand Функция для создания новой команды для commandProcessor.
//
// commandType - указывается тип команды models.CmdtypeTYPE_* в зависимости от требуемой логики.
// scope - по умолчанию указывается models.CmdscopeSCOPE_ALL_UNSPECIFIED.
// Для обработки конкретного скрипта указывать models.CmdscopeSCOPE_ID, scriptIDs и increaseRPS
// Если указан models.CmdscopeSCOPE_ID, то требуется указать массив scriptIDs.
// percentageOfTarget - поле требуется в командах на корректировку нагрузки в процентах,
// по умолчанию указывать "0" или 100,
// если указать > 0 скрипты будут запускаться с рассчитанным RPS`ом по этому полю
// runID - команда всегда обрабатывает только один конкретный Run.
// increaseRPS - увеличение на конкретное значение
func (s *Store) createMCommand(ctx context.Context,
	commandType string, scope string, runID int32, scriptIDs []int64, percentageOfTarget int32, increaseRPS int32) (
	mCmd *models.Command, err error) {

	logger.Infof(ctx,
		"Creating new command('%v'; scope: '%v'; runID: '%v'; scriptIDs: '%v'; percentageOfTarget: '%v';)",
		commandType, scope, runID, scriptIDs, percentageOfTarget)

	mCmd = &models.Command{
		Type:               commandType,
		Scope:              scope,
		RunID:              runID,
		Status:             models.CmdstatusSTATUS_CREATED_UNSPECIFIED,
		Hostname:           config.Get(ctx).Hostname,
		ScriptIds:          scriptIDs,
		PercentageOfTarget: null.Int32From(percentageOfTarget),
		IncreaseRPS:        increaseRPS,
	}

	err = mCmd.Insert(ctx, s.db, boil.Blacklist(
		models.CommandColumns.DeletedAt,
		models.CommandColumns.CommandID,
	))
	if err != nil {
		err = errors.Wrap(err, "error when writing a command to the database")
		logger.Errorf(ctx, "mCmd.Insert: %v", err)

		return nil, err
	}

	logger.Infof(ctx, "Insert command success ID:'%+v'", mCmd.CommandID)
	logger.Infof(ctx, "Created mCommand:'%+v'", mCmd)

	return mCmd, err
}

func (s *Store) NewMCmdStopScript(ctx context.Context, runID int32, scriptIDs []int64) (
	pbCommand *models.Command, err error) {
	return s.createMCommand(ctx, models.CmdtypeTYPE_STOP_SCRIPT,
		models.CmdscopeSCOPE_ID, runID, scriptIDs, 0, -1)
}

func (s *Store) updateMCommand(ctx context.Context, cmd *models.Command) bool {
	if cmd.ErrorDescription != "" { //fixme: не делать логики в слое БД!!!!
		cmd.ErrorDescription = fmt.Sprintf("%v", fmt.Sprintf("%.250s...", cmd.ErrorDescription))
	}

	_, err := cmd.Update(ctx, s.db, boil.Blacklist(
		models.CommandColumns.DeletedAt,
	))
	if err != nil {
		logger.Errorf(ctx, "Error updateMCommand(CommandID:'%v' ERROR:'%+v')", cmd.CommandID, err)
		cmd.ErrorDescription = "error updating the status command"

		return false
	}

	return true
}

// GetMCommandToProcess Функция получения команды для обработки КомандПроцессором
func (s *Store) GetMCommandToProcess(ctx context.Context, hostname string) (cmd *models.Command, status bool) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "GetMCommandToProcess failed: '%+v'", err)
		}
	}()

	if config.Get(ctx).ENV != config.EnvInfra {
		defer undecided.InfoTimer(ctx, "GetMCommandToProcess")()
	}

	cmd, err := s.getPriorityMCommand(ctx, hostname)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			logger.Infof(ctx, "Select 'Command' no rows in result set")
		} else {
			logger.Errorf(ctx, "Select 'Command' ERROR:'%+v'", err)
		}

		return cmd, false
	}

	logger.Infof(ctx, "Command returning '%v', '%v', '%+v', ", cmd.Type, cmd.CommandID, cmd)

	return cmd, true
}

//	getPriorityMCommand Проверяем какие команды нужно запросить в порядке приоритета
//
// -> TYPE_STOP_SCENARIO, TYPE_STOP_SCRIPT, TYPE_RUN_SCRIPT, TYPE_RUN_SIMPLE_SCRIPT, TYPE_RUN_SCENARIO, TYPE_ADJUSTMENT, TYPE_INCREASE ->
// потом все остальные(CmdtypeTYPE_UPDATE)
func (s *Store) getPriorityMCommand(ctx context.Context, hostname string) (cmd *models.Command, err error) {
	qms := append([]qm.QueryMod{},
		models.CommandWhere.Status.EQ(models.CmdstatusSTATUS_CREATED_UNSPECIFIED),
	)

	// в infra всегда один под, по этому не ограничивать выборку по хосту
	if config.Get(ctx).ENV != config.EnvInfra {
		qms = append(qms,
			models.CommandWhere.Hostname.EQ(hostname),
		)
	}

	var exists bool
	// sequential search for commands by priority
	if exists, err = s.checkingForThePresenceOfACommandByTheType(ctx,
		models.CmdtypeTYPE_STOP_SCENARIO, &qms); exists {
		logger.Infof(ctx, "Exists command type: '%v'", models.CmdtypeTYPE_STOP_SCENARIO)
		qms = append(qms,
			models.CommandWhere.Type.EQ(models.CmdtypeTYPE_STOP_SCENARIO),
		)
	} else if exists, err = s.checkingForThePresenceOfACommandByTheType(ctx,
		models.CmdtypeTYPE_STOP_SCRIPT, &qms); exists {
		logger.Infof(ctx, "Exists command type: '%v'", models.CmdtypeTYPE_STOP_SCRIPT)
		qms = append(qms,
			models.CommandWhere.Type.EQ(models.CmdtypeTYPE_STOP_SCRIPT),
		)
	} else if exists, err = s.checkingForThePresenceOfACommandByTheType(ctx,
		models.CmdtypeTYPE_RUN_SCRIPT, &qms); exists {
		logger.Infof(ctx, "Exists command type: '%v'", models.CmdtypeTYPE_RUN_SCRIPT)
		qms = append(qms,
			models.CommandWhere.Type.EQ(models.CmdtypeTYPE_RUN_SCRIPT),
		)
	} else if exists, err = s.checkingForThePresenceOfACommandByTheType(ctx,
		models.CmdtypeTYPE_RUN_SIMPLE_SCRIPT, &qms); exists {
		logger.Infof(ctx, "Exists command type: '%v'", models.CmdtypeTYPE_RUN_SIMPLE_SCRIPT)
		qms = append(qms,
			models.CommandWhere.Type.EQ(models.CmdtypeTYPE_RUN_SIMPLE_SCRIPT),
		)
	} else if exists, err = s.checkingForThePresenceOfACommandByTheType(ctx,
		models.CmdtypeTYPE_RUN_SCENARIO_UNSPECIFIED, &qms); exists {
		logger.Infof(ctx, "Exists command type: '%v'", models.CmdtypeTYPE_RUN_SCENARIO_UNSPECIFIED)
		qms = append(qms,
			models.CommandWhere.Type.EQ(models.CmdtypeTYPE_RUN_SCENARIO_UNSPECIFIED),
		)
	} else if exists, err = s.checkingForThePresenceOfACommandByTheType(ctx,
		models.CmdtypeTYPE_ADJUSTMENT, &qms); exists {
		logger.Infof(ctx, "Exists command type: '%v'", models.CmdtypeTYPE_ADJUSTMENT)
		qms = append(qms,
			models.CommandWhere.Type.EQ(models.CmdtypeTYPE_ADJUSTMENT),
		)
	} else if exists, err = s.checkingForThePresenceOfACommandByTheType(ctx,
		models.CmdtypeTYPE_INCREASE, &qms); exists {
		logger.Infof(ctx, "Exists command type: '%v'", models.CmdtypeTYPE_INCREASE)
		qms = append(qms,
			models.CommandWhere.Type.EQ(models.CmdtypeTYPE_INCREASE),
		)
	} else {
		logger.Info(ctx, "__--__--No priority commands found--__--__")
	}

	if err != nil {
		logger.Errorf(ctx, "GetMCommandToProcess 'exists error'='%v'", err)
	}

	cmd, err = models.Commands(append(qms, qm.OrderBy(models.CommandColumns.CommandID))...).One(ctx, s.db)
	if cmd != nil {
		s.UpdateStatusMCommand(ctx, cmd, models.CmdstatusSTATUS_PROCESSED, "")
	}

	return cmd, err
}

func (s *Store) CountNewCmd(ctx context.Context, hostname string) (int64, error) {
	countNewCmd, er := models.Commands(
		models.CommandWhere.Status.EQ(models.CmdstatusSTATUS_CREATED_UNSPECIFIED),
		models.CommandWhere.Hostname.EQ(hostname),
	).Count(ctx, s.db)

	return countNewCmd, er
}

func (s *Store) UpdateStatusMCommand(
	ctx context.Context,
	cmd *models.Command,
	status string,
	errorMessage string,
) bool {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "UpdateStatusMCommand failed: '%+v'", err)
		}
	}()

	if status != models.CmdstatusSTATUS_COMPLETED {
		logger.Warnf(
			ctx,
			"Update Status MCommand:{ type:{%v}, RunID{%v},  message:{%v}}",
			cmd.Type,
			cmd.RunID,
			errorMessage,
		)
		if status == models.CmdstatusSTATUS_FAILED {
			logger.Errorf(
				ctx,
				"Update Status{%v} MCommand:{ type:{%v}, RunID{%v},  message:{%v}}",
				status,
				cmd.Type,
				cmd.RunID,
				errorMessage,
			)
		}
	}

	if errorMessage != "" {
		if cmd.ErrorDescription != "" {
			cmd.ErrorDescription = fmt.Sprintf("%v; %v", cmd.ErrorDescription, errorMessage)
		} else {
			cmd.ErrorDescription = errorMessage
		}
	}

	cmd.Status = status

	return s.updateMCommand(context.WithoutCancel(ctx), cmd)
}

func (s *Store) DeleteMCmd(ctx context.Context, cmd *models.Command) bool {
	if cmd.Status == models.CmdstatusSTATUS_COMPLETED && cmd.ErrorDescription == "" {
		_, err := cmd.Delete(ctx, s.db)
		if err != nil {
			logger.Warnf(ctx, "DeleteMCmd: RunID:'%v' error:'%+v'", cmd.RunID, err)
			cmd.ErrorDescription = err.Error()

			return false
		}
	} else {
		logger.Warnf(ctx, "CMD does not meet the criteria for deletion: {Status:'%v'; ErrorDescription:'%v'}",
			cmd.Status, cmd.ErrorDescription)
	}

	return true
}

//	StopObservingForStatusRun Удаление из таблицы очереди команд, команды обновления статуса Рана по RunID.
//
// Удаляет тип команды UPDATE по RunID.
// Требуется для разгрузки очереди, к примеру когда запуск останавливается - нет смысла выполнять обновление этого запуска.
// fixme: эта функция должна быть в другом месте, туда же нужно перенести и функцию job.observingForStatusRun()
func (s *Store) StopObservingForStatusRun(ctx context.Context, runID int32) bool {
	qms := append([]qm.QueryMod{},
		models.CommandWhere.Type.EQ(models.CmdtypeTYPE_UPDATE),
	)

	exists, err := s.checkingForThePresenceOfACommandByTheRunID(ctx, runID, qms)
	if err != nil {
		logger.Warnf(ctx, "error checking the availability of commands in the database{Error: %v}", err)
		return false
	}

	var cmd *models.Command

	if exists {
		logger.Infof(ctx, "Exist command type '%v' for the Run %v", models.CmdtypeTYPE_UPDATE, runID)

		cmd, err = models.Commands(
			models.CommandWhere.Type.EQ(models.CmdtypeTYPE_UPDATE),
			models.CommandWhere.RunID.EQ(runID),
			models.CommandWhere.Status.EQ(models.CmdstatusSTATUS_CREATED_UNSPECIFIED),
		).One(ctx, s.db)
		if err != nil {
			logger.Warnf(ctx, "error extracting a command from the database{Error: %v}", err)
			return false
		}

		logger.Warnf(ctx,
			"The redundant update command will be deleted. CommandID(%v), RunID(%v), Status(%v), ErrorDescription(%v)",
			cmd.CommandID, cmd.RunID, cmd.Status, cmd.ErrorDescription)

		return s.DeleteMCmd(ctx, cmd)
	}

	return true
}

//	checkingForThePresenceOfACommandByTheType Метод для проверки наличия команды по типу команды
//
// qms - дополняет указанным типом массив QueryMod,
// используется для ограничения выполнения запроса, на получение найденной команды
func (s *Store) checkingForThePresenceOfACommandByTheType(ctx context.Context, cmdType string, qms *[]qm.QueryMod) (
	bool, error) {
	return models.Commands(
		append(*qms, models.CommandWhere.Type.EQ(cmdType))...,
	).Exists(ctx, s.db)
}

//	checkingForThePresenceOfACommandByTheRunID Метод для проверки наличия команды по RunID
//
// qms - дополняет указанным типом массив QueryMod,
// используется для ограничения выполнения запроса, на получение найденной команды
// nolint
func (s *Store) checkingForThePresenceOfACommandByTheRunID(ctx context.Context, runID int32, qms []qm.QueryMod) (
	bool, error) {
	return models.Commands(
		append(qms,
			models.CommandWhere.RunID.EQ(runID),
			models.CommandWhere.Status.EQ(models.CmdstatusSTATUS_CREATED_UNSPECIFIED),
		)...,
	).Exists(ctx, s.db)
}

func (s *Store) NewMCmdUpdate(ctx context.Context, runID int32) (
	pbCommand *models.Command, err error) {
	return s.createMCommand(ctx, models.CmdtypeTYPE_UPDATE,
		models.CmdscopeSCOPE_ALL_UNSPECIFIED, runID, nil, 0, -1)
}

func (s *Store) NewMCmdStopScenario(ctx context.Context, runID int32) (
	pbCommand *models.Command, err error) {
	return s.createMCommand(ctx, models.CmdtypeTYPE_STOP_SCENARIO,
		models.CmdscopeSCOPE_ALL_UNSPECIFIED, runID, nil, 0, -1)
}

func (s *Store) NewMCmdRunScenarioWithRpsAdjustment(ctx context.Context, runID int32, percentageOfTarget int32) (
	pbCommand *models.Command, err error) {
	return s.createMCommand(ctx, models.CmdtypeTYPE_RUN_SCENARIO_UNSPECIFIED,
		models.CmdscopeSCOPE_ALL_UNSPECIFIED, runID, nil, percentageOfTarget, -1)
}

func (s *Store) NewMCmdRunScriptWithRpsAdjustment(ctx context.Context,
	runID int32, scriptIDs []int64, percentageOfTarget int32) (
	pbCommand *models.Command, err error) {
	return s.createMCommand(ctx, models.CmdtypeTYPE_RUN_SCRIPT,
		models.CmdscopeSCOPE_ID, runID, scriptIDs, percentageOfTarget, -1)
}

func (s *Store) NewMCmdRunScript(ctx context.Context, runID int32, scriptIDs []int64) (
	pbCommand *models.Command, err error) {
	return s.NewMCmdRunScriptWithRpsAdjustment(ctx, runID, scriptIDs, 0)
}

func (s *Store) NewMCmdAdjustment(ctx context.Context, runID int32, percentageOfTarget int32) (
	pbCommand *models.Command, err error) {
	return s.createMCommand(ctx, models.CmdtypeTYPE_ADJUSTMENT,
		models.CmdscopeSCOPE_ALL_UNSPECIFIED, runID, nil, percentageOfTarget, -1)
}

func (s *Store) NewMCmdRunSimpleScriptWithRpsAdjustment(ctx context.Context,
	runID int32, scriptIDs []int64, percentageOfTarget int32) (
	pbCommand *models.Command, err error) {
	return s.createMCommand(ctx, models.CmdtypeTYPE_RUN_SIMPLE_SCRIPT,
		models.CmdscopeSCOPE_ID, runID, scriptIDs, percentageOfTarget, -1)
}

func (s *Store) NewMCmdRunSimpleScript(ctx context.Context, runID int32, scriptIDs []int64) (
	pbCommand *models.Command, err error) {
	return s.NewMCmdRunSimpleScriptWithRpsAdjustment(ctx, runID, scriptIDs, 0)
}

func (s *Store) GetRunCommand(ctx context.Context, runID int32) (returnRun *models.Command, err error) {
	mRun, err := models.Runs(
		models.RunWhere.RunID.EQ(runID),
	).One(ctx, s.db)
	if err != nil {
		err = errors.Wrapf(err, "Error fetch runs: '%v'", runID)
		logger.Error(ctx, "Errorr: ", err)
		return nil, err
	}

	return mRun.Commands(qm.OrderBy(fmt.Sprint(models.CommandColumns.CommandID, " desc"))).One(ctx, s.db)
}
