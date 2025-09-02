package util

import (
	"context"
	"strings"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"golang.org/x/exp/slices"
)

func TrimSpaceInScript(script *pb.Script) {
	script.ScriptFile = strings.TrimSpace(script.GetScriptFile())
	script.BaseUrl = strings.TrimSpace(script.GetBaseUrl())
	script.AmmoId = strings.TrimSpace(script.GetAmmoId())
	script.Options.Duration = strings.TrimSpace(script.GetOptions().GetDuration())
	script.Options.Steps = strings.TrimSpace(script.GetOptions().GetSteps())
	script.Options.Rps = strings.TrimSpace(script.GetOptions().GetRps())
	script.Name = strings.TrimSpace(script.Name)
}

func TrimSpaceInSimpleScript(simpleScript *pb.SimpleScript) {
	simpleScript.ScriptFileUrl = strings.TrimSpace(simpleScript.GetScriptFileUrl())
	simpleScript.Tag = strings.TrimSpace(simpleScript.Tag)
	simpleScript.HttpMethod = strings.TrimSpace(simpleScript.HttpMethod)
	simpleScript.Scheme = strings.TrimSpace(simpleScript.Scheme)
	simpleScript.StaticAmmo = strings.TrimSpace(simpleScript.StaticAmmo)
	simpleScript.Path = strings.TrimSpace(simpleScript.Path)
	simpleScript.AmmoUrl = strings.TrimSpace(simpleScript.AmmoUrl)
	simpleScript.Steps = strings.TrimSpace(simpleScript.Steps)
	simpleScript.MaxVUs = strings.TrimSpace(simpleScript.MaxVUs)
	simpleScript.Duration = strings.TrimSpace(simpleScript.Duration)
	simpleScript.Name = strings.TrimSpace(simpleScript.Name)
	simpleScript.Rps = strings.TrimSpace(simpleScript.Rps)

	for i := range simpleScript.QueryParams {
		simpleScript.QueryParams[i].Key = strings.TrimSpace(simpleScript.QueryParams[i].Key)
		simpleScript.QueryParams[i].Value = strings.TrimSpace(simpleScript.QueryParams[i].Value)
	}
}

// GetArrayActiveScriptIDs возвращает только идентификаторы заенейбленных скриптов по scenarioID
func GetArrayActiveScriptIDs(
	ctx context.Context,
	scenarioID int32,
	db *data.Store,
) (arrScriptID []int32, message string) {
	mScripts, er := db.GetAllEnabledMScripts(ctx, scenarioID)
	if er != nil {
		return nil, er.Error()
	}

	for _, mScript := range mScripts {
		arrScriptID = append(arrScriptID, mScript.ScriptID)
	}

	return arrScriptID, ""
}

// GetArrayActiveSimpleScriptIDs возвращает только идентификаторы заенейбленных Simple скриптов по scenarioID
func GetArrayActiveSimpleScriptIDs(
	ctx context.Context,
	scenarioID int32,
	db *data.Store,
) (arrSimpleScriptID []int32, message string) {
	mSimpleScripts, er := db.GetAllEnabledMSimpleScripts(ctx, scenarioID)
	if er != nil {
		return nil, er.Error()
	}

	for _, mScript := range mSimpleScripts {
		arrSimpleScriptID = append(arrSimpleScriptID, mScript.ScriptID)
	}

	return arrSimpleScriptID, ""
}

// CheckingNegativeValue Проверка на отрицательное числовое значение.
// Возвращает всегда положительное представление number
func CheckingNegativeValue(number string) string {
	if number[0:1] == "-" {
		return number[1:]
	}

	return number
}

// CheckingStaticAmmoLength Проверка длинны статического тела, что бы не хранить большие тела в базе
func CheckingStaticAmmoLength(ctx context.Context, lenStaticAmmo int) bool {
	return config.Get(ctx).MaxStaticAmmoLength >= lenStaticAmmo
}

var timeUnits = []string{"s", "m", "h", "d"}

func IsTimeUnit(tu string) bool {
	return slices.Contains(timeUnits, tu)
}
