package job

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	agentapi2 "github.com/aliexpressru/alilo-backend/pkg/model/agentapi"
	"github.com/aliexpressru/alilo-backend/pkg/util/httputil"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	mathutil "github.com/aliexpressru/alilo-backend/pkg/util/math"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

const (
	statisticTrackingContextKey = "_statistic_tracker"
)

var (
	dStat               time.Duration
	traceGetterHostname = ""
)

func StatisticTracker(ctx context.Context, p *ProcessorPool) {
	ctx = undecided.NewContextWithMarker(ctx, statisticTrackingContextKey, "")
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "StatisticTracker failed: '%+v'", err)
		}
	}()
	if traceGetterHostname == "" {
		traceGetterHostname = config.Get(ctx).TraceGetterHostname
	}
	cfg := config.Get(ctx)
	dStat = cfg.JobStatisticTrackingFrequency
	for range time.Tick(dStat) {
		logger.Info(ctx, "Start StatisticTracker")
		countRunsByStatusRunning, err := p.db.GetCountRunsByStatus(ctx, pb.Run_STATUS_RUNNING)
		if err != nil {
			logger.Warnf(ctx, "Error getting the number of running tests: %v", err)

			continue
		}
		logger.Infof(ctx, "There are running scenarios{%v}, we start collecting statistics", countRunsByStatusRunning)

		if countRunsByStatusRunning > 0 {
			logger.Info(ctx, "Make a collection")
			ctx = undecided.NewContextWithMarker(ctx, statisticTrackingContextKey, "collecting")
			p.CollectingDump(ctx)
		}
	}
}

type dumpStat struct {
	sync.RWMutex
	ID      int32
	mapStat map[int64]*models.Statistic
	traces  []*models.Trace
}

func (ds *dumpStat) appendStatus(ctx context.Context, resp *agentapi2.ResponseGetStatus, agentHostName string) {
	if resp == nil || resp.Metrics == nil {
		return
	}

	ds.Lock()
	defer ds.Unlock()

	// формируем ключ по Task
	if resp.Task == nil {
		logger.Warn(ctx, "appendStatus called without Task")
		return
	}

	//key := fmt.Sprintf("%s:%s", resp.Task.Method, resp.Task.Path)
	key := resp.Task.Pid

	if statVal, ok := ds.mapStat[key]; !ok {
		// если записи ещё нет → создаём
		curStat := &models.Statistic{
			StatisticDumpID: ds.ID,
			//URLMethod:       resp.Task.Method,
			//URLPath:         resp.Task.Path,

			//ProjectIds:  types.Int64Array{resp.Task.ProjectID},
			//ScenarioIds: types.Int64Array{resp.Task.ScenarioID},
			//ScriptIds:   types.Int64Array{resp.Task.ScriptID},
			//RunIds:      types.Int64Array{resp.Task.RunID},
			// ScriptRunIds — если есть в Task
			//ScriptRunIds: types.Int64Array{resp.Task.ScriptRunID},

			RPS:   mathutil.Int32Fm(resp.Metrics.Rps),
			RTMax: mathutil.Int32Fm(resp.Metrics.RtMax),
			RT90P: mathutil.Int32Fm(resp.Metrics.Rt90P),
			RT95P: mathutil.Int32Fm(resp.Metrics.Rt95P),
			RT99P: mathutil.Int32Fm(resp.Metrics.Rt99P),

			//nolint:gosec
			Failed: mathutil.Int32Fm(resp.Metrics.Failed),
			Vus:    mathutil.Int32Fm(resp.Metrics.Vus),

			DataSent:     mathutil.Int32Fm(resp.Metrics.Sent),
			DataReceived: mathutil.Int32Fm(resp.Metrics.Received),

			//CurrentTestRunDuration: strconv.FormatInt(resp.Metrics.CurrentTestRunDuration.AsDuration().Nanoseconds(), 10),
			Agents: []string{agentHostName},
		}
		ds.mapStat[key] = curStat
	} else {
		// если запись есть → объединяем
		statVal.RPS += mathutil.Int32Fm(resp.Metrics.Rps)
		statVal.RTMax = max(statVal.RTMax, mathutil.Int32Fm(resp.Metrics.RtMax))
		statVal.RT90P = max(statVal.RT90P, mathutil.Int32Fm(resp.Metrics.Rt90P))
		statVal.RT95P = max(statVal.RT95P, mathutil.Int32Fm(resp.Metrics.Rt95P))
		statVal.RT99P = max(statVal.RT99P, mathutil.Int32Fm(resp.Metrics.Rt99P))

		//nolint:gosec
		//statVal.Failed += (resp.Metrics.Failed)

		statVal.Vus += mathutil.Int32Fm(resp.Metrics.Vus)
		statVal.DataSent += mathutil.Int32Fm(resp.Metrics.Sent)
		statVal.DataReceived += mathutil.Int32Fm(resp.Metrics.Received)

		// апдейт продолжительности (берём макс)
		//statVal.CurrentTestRunDuration = strconv.FormatInt(
		//	max(
		//		mathutil.Int64Fm(statVal.CurrentTestRunDuration),
		//		resp.Metrics.CurrentTestRunDuration.AsDuration().Nanoseconds(),
		//	),
		//	10,
		//)

		// если надо — добавляем Project/Scenario/Script/RunID как в grpc-версии
		//if !containsInt64(statVal.ProjectIds, resp.Task.ProjectID) {
		//	statVal.ProjectIds = append(statVal.ProjectIds, resp.Task.ProjectID)
		//}
		//if !containsInt64(statVal.ScenarioIds, resp.Task.ScenarioID) {
		//	statVal.ScenarioIds = append(statVal.ScenarioIds, resp.Task.ScenarioID)
		//}
		//if !containsInt64(statVal.ScriptIds, resp.Task.ScriptID) {
		//	statVal.ScriptIds = append(statVal.ScriptIds, resp.Task.ScriptID)
		//}
		//if !containsInt64(statVal.RunIds, resp.Task.RunID) {
		//	statVal.RunIds = append(statVal.RunIds, resp.Task.RunID)
		//}
		//if resp.Task.ScriptRunID > 0 && !containsInt64(statVal.ScriptRunIds, resp.Task.ScriptRunID) {
		//	statVal.ScriptRunIds = append(statVal.ScriptRunIds, resp.Task.ScriptRunID)
		//}

		// добавляем имя агента, если ещё не было
		//if !containsString(statVal.Agents, agentHostName) {
		//	statVal.Agents = append(statVal.Agents, agentHostName)
		//}
	}
}

func (p *ProcessorPool) CollectingDump(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "CollectingDump failed: '%+v'", err)
		}
	}()

	defer undecided.InfoTimer(ctx, "CollectingDump")()

	mStatisticDump, err := p.db.CreateMStatisticDump(ctx)
	if err != nil {
		logger.Errorf(ctx, "CreateMStatisticDump{%v} failed: '%+v'", mStatisticDump.StatisticDumpID, err)

		return
	}
	logger.Infof(ctx, "CreateMStatisticDump ID{%v}", mStatisticDump.StatisticDumpID)
	statisticsD := &dumpStat{
		mapStat: make(map[int64]*models.Statistic, 0),
		ID:      mStatisticDump.StatisticDumpID,
	}

	dumpWaitGroup := &sync.WaitGroup{}
	agents, err := p.db.GetAllEnabledMAgents(ctx)
	if err != nil {
		logger.Errorf(ctx, "CollectingDump{%v} error Getting all agents: '%+v'", statisticsD.ID, err)
	}
	logger.Infof(ctx, "Dump{%v}. agents count: %v", statisticsD.ID, len(agents))
	for i, agent := range agents {
		dumpWaitGroup.Add(1)
		logger.Infof(ctx, "Dump{%v} (%v) %+v", statisticsD.ID, i, agent)

		execPool.Go(func() { p.collectingStatistics(ctx, agent, statisticsD, dumpWaitGroup) })
	}
	logger.Infof(ctx, "dumpGroup waiting{%v}...", statisticsD.ID)
	dumpWaitGroup.Wait()
	logger.Infof(ctx, "dumpGroup waited{%v}", statisticsD.ID)

	if len(statisticsD.mapStat) > 0 {
		logger.Infof(ctx, "Load statID{%v}, traces{%v}", statisticsD.ID, len(statisticsD.traces))
		logger.Infof(ctx, "Save the collected statistics{%v}", statisticsD.ID)
		for key, statistic := range statisticsD.mapStat {
			logger.Debugf(ctx, "Save the collected{%v} stat{%v}", statistic.StatisticDumpID, key)
			err = p.db.PutMStatistic(ctx, statistic)
			if err != nil {
				logger.Errorf(ctx, "Error Put statistic{%v:%v}: '%v'", statistic.StatisticDumpID, key, err.Error())
			}
		}

		logger.Infof(ctx, "Save the collected{%v} traces len{%v}", statisticsD.ID, len(statisticsD.traces))
		err = p.db.PutMTraces(ctx, statisticsD.traces)
		if err != nil {
			logger.Errorf(ctx, "Error Put traces{%v:%v}: '%v'", statisticsD.ID, len(statisticsD.traces), err.Error())
		}
	} else {
		logger.Infof(ctx, "Statistics do not have data to save")
		err = p.db.DeleteMStatisticDump(ctx, statisticsD.ID)
		if err != nil {
			logger.Errorf(ctx, "Error delete MDump: '%v'", err.Error())
		}
	}
}

func (p *ProcessorPool) collectingStatistics(ctx context.Context,
	agent *models.Agent, dumpS *dumpStat, group *sync.WaitGroup) {
	defer func(group *sync.WaitGroup) {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "collecting statistics failed: '%+v'", err)
		}

		group.Done()
	}(group)

	defer undecided.InfoTimer(ctx, fmt.Sprintf("CollectingStatistics{%v}", agent.HostName))()

	url := fmt.Sprintf("http://%s:%s", agent.HostName, agent.Port)

	b, err := httputil.Get(ctx, fmt.Sprintf("%s/api/v1/getAllTasks", url))
	if err != nil {
		logger.Errorf(ctx, "get all tasks failed: '%+v'", err)
		return
	}
	tasks := &agentapi2.ResponseGetAllTasks{}
	err = json.Unmarshal(b, tasks)
	if err != nil {
		logger.Errorf(ctx, "failed unmarshal get all tasks response: '%+v'", err)
		return
	}
	for pid := range tasks.Tasks {
		var bytes []byte
		bytes, err = httputil.Post(
			ctx,
			fmt.Sprintf("%s/api/v1/getStatus", url),
			"application/json",
			map[string]string{}, &agentapi2.GetStatusRequest{Pid: pid})
		if err != nil {
			logger.Errorf(ctx, "HTTP Start request failed: %v", err)
			return
		}

		status := &agentapi2.ResponseGetStatus{}

		err = json.Unmarshal(bytes, status)
		if err != nil {
			logger.Errorf(ctx, "HTTP get status request failed: %v", err)

			return
		}

		dumpS.appendStatus(ctx, status, agent.HostName)
	}
	logger.Infof(ctx, "Total statistics{%s} count statistics{%v}", agent.HostName, len(dumpS.mapStat))
}
