package agentapi

import (
	"os"
	"os/exec"
	"time"
)

// nolint:unused
const (
	ResponseStatusSuccess  string = "Success"
	ResponseStatusError    string = "Error"
	ResponseStatusKill     string = "exit status 0"
	ResponseStatusStopped  string = "exit status 105"
	ResponseStatusStopping string = "signal: killed"

	ResponseStatusDefunct          string = "the process is defunct"
	ResponseStatusNoSuchTask       string = "there is no such task"
	ResponseStatusNoPortsAvailable string = "no ports available"
)

type AgentMetricsResponse struct {
	Status           string           `json:"status"`
	Error            string           `json:"error"`
	AgentUtilization AgentUtilization `json:"agentUtilization"`
}

type AgentUtilization struct {
	CPUUsed   int `json:"cpuUsed"`
	MemUsed   int `json:"memUsed"`
	PortsUsed int `json:"portsUsed"`
}

type AgentStartRequest struct {
	ScenarioTitle string   `json:"scenarioTitle"`
	ScriptTitle   string   `json:"scriptTitle"`
	ScriptURL     string   `json:"scriptURL"`
	AmmoURL       string   `json:"ammoURL"`
	Params        []string `json:"params"`
}

type Task struct {
	Pid            int64     `json:"pid"`
	Path           string    `json:"path"`
	ScenarioTitle  string    `json:"scenarioTitle"`
	Method         string    `json:"method"`
	ScriptTitle    string    `json:"scriptTitle"`
	ScriptURL      string    `json:"scriptURL"`
	AmmoURL        string    `json:"ammoURL"`
	ScriptFileName string    `json:"scriptFileName"`
	LogFileName    string    `json:"logFileName"`
	K6ApiPort      string    `json:"k6ApiPort"`
	PortPrometheus string    `json:"portPrometheus"`
	Params         []string  `json:"params"`
	Cmd            *exec.Cmd `json:"-"`
	LogFile        *os.File  `json:"-"`
	StartTime      time.Time `json:"startTime"`
}

type Metrics struct {
	Rps      string `json:"rps,omitempty"`
	Rt90P    string `json:"rt90p,omitempty"`
	Rt95P    string `json:"rt95p,omitempty"`
	RtMax    string `json:"rtMax,omitempty"`
	Rt99P    string `json:"rt99p,omitempty"`
	Failed   string `json:"failed,omitempty"`
	Vus      string `json:"vus,omitempty"`
	Sent     string `json:"sent,omitempty"`
	Received string `json:"received,omitempty"`
}

type GetStatusRequest struct {
	Pid int64 `json:"pid"`
}

type ResponseGetAllTasks struct {
	Status string          `json:"status"`
	Error  string          `json:"error"`
	Tasks  map[int64]*Task `json:"tasks"`
}

type ResponseGetStatus struct {
	Status  string   `json:"status"`
	Error   string   `json:"error"`
	Task    *Task    `json:"task"`
	Metrics *Metrics `json:"metrics"`
}
