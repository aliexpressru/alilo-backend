package util

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

type SourceScript struct {
	Imports         []Import
	Lets            []Let
	Options         Options
	DefaultFunction DefaultFunction
}

type Import struct {
	Import string
}

type Let struct {
	Let string
}

type Options struct {
	SummaryTrendStats     []string
	InsecureSkipTLSVerify bool
	DiscardResponseBodies bool
	Scenarios             []Scenario
}

type Scenario struct {
	Name            string
	Executor        string
	StartRate       int32
	TimeUnit        string
	PreAllocatedVUs int32
	MaxVUs          int32
	Stages          string
}

type DefaultFunction struct {
	Constants []string
	Lets      []Let
	Group     string
	Rq        Request
}

type Request struct {
	Type          string
	URL           string
	IsQueryParams bool
	QueryParams   map[string]string
	IsParams      bool
	IsPayload     bool
}

func (o Options) String() string {
	defer undecided.DebugTimer(undecided.NewContextWithMarker(context.Background(), "_util", ""), "Options to String")()

	result := strings.Builder{}
	result.WriteString("export const options = {")

	trends := "\n\tsummaryTrendStats: ["
	for i, trend := range o.SummaryTrendStats {
		trends = fmt.Sprintf("%v\"%v\"", trends, trend)
		if i < len(o.SummaryTrendStats)-1 {
			trends = fmt.Sprint(trend, ", ")
		}
	}

	result.WriteString(trends)
	result.WriteString("]")

	result.WriteString(",\n\tinsecureSkipTLSVerify: ")
	result.WriteString(strconv.FormatBool(o.InsecureSkipTLSVerify))
	result.WriteString(",\n\tdiscardResponseBodies: ")
	result.WriteString(strconv.FormatBool(o.DiscardResponseBodies))

	result.WriteString(",\n\tscenarios: {")

	for _, scenario := range o.Scenarios {
		result.WriteString("\n\t\t")
		result.WriteString(scenario.Name)
		result.WriteString(": {")
		result.WriteString("\n\t\t\texecutor: '")
		result.WriteString(scenario.Executor)
		result.WriteString("'")
		result.WriteString(",\n\t\t\tstartRate: ")
		result.WriteString(strconv.FormatInt(int64(scenario.StartRate), 10))
		result.WriteString(",\n\t\t\ttimeUnit: '")
		result.WriteString(scenario.TimeUnit)
		result.WriteString("'")
		result.WriteString(",\n\t\t\tpreAllocatedVUs: ")
		result.WriteString(strconv.FormatInt(int64(scenario.PreAllocatedVUs), 10))
		result.WriteString(",\n\t\t\tmaxVUs: ")
		result.WriteString(strconv.FormatInt(int64(scenario.MaxVUs), 10))
		result.WriteString(",\n\t\t\tstages: ")
		result.WriteString(scenario.Stages)

		result.WriteString("\n\t\t}")
	}

	result.WriteString("\n\t}\n}")

	return result.String()
}

func (d DefaultFunction) String() string {
	result := strings.Builder{}
	result.WriteString("export default function () {")

	for _, s := range d.Lets {
		result.WriteString("\n\t")
		result.WriteString(s.Let)
	}

	result.WriteString("\n\tgroup('")
	result.WriteString(d.Group)
	result.WriteString("', function () {")
	result.WriteString("\n\t\tres = http.get(\"")
	result.WriteString(d.Rq.URL)
	result.WriteString("\"")

	if d.Rq.IsQueryParams {
		result.WriteString("?")
	}

	if d.Rq.IsPayload {
		result.WriteString(", JSON.stringify(payload)")
	}

	if d.Rq.IsParams {
		result.WriteString(", params()")
	}

	result.WriteString("\");")
	result.WriteString("\n\t})")
	result.WriteString("\n\t// Validate response status\n\t")
	result.WriteString("check(res, { \"status was 200\": (r) => r.status === 200 });\n\t")
	result.WriteString("sleep(1);")

	result.WriteString("\n}")

	return result.String()
}
