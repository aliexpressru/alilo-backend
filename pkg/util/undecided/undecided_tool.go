package undecided //	fixme: разнести функции

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"go.uber.org/zap"
)

// var logger = zap.S()

func GetPBHost(agent *pb.Agent) string {
	return fmt.Sprintf("%v:%v", agent.HostName, agent.Port)
}

func GetMHost(agent *models.Agent) string {
	return fmt.Sprintf("%v:%v", agent.HostName, agent.Port)
}

func GetHost(host string, port string) string {
	return fmt.Sprintf("%v:%v", host, port)
}

func AppendDateTime(AppendedString string) string {
	const Layout = "2006_01_02_15_04_05_"

	t := time.Now()

	return t.Format(fmt.Sprint(Layout, AppendedString))
}

func AppendDate(AppendedString string) string {
	const Layout = "2006_01_02_"

	t := time.Now()

	return t.Format(fmt.Sprint(Layout, AppendedString))
}

func MakePath(elements []string) string {
	path := strings.Join(elements, "/")

	return path
}

func ConvertArrayInt32toArrayInt64(arrayInt32 []int32) (arrayInt64 []int64) {
	for _, i32 := range arrayInt32 {
		arrayInt64 = append(arrayInt64, int64(i32))
	}

	return arrayInt64
}

func ConvInt32toArrInt64(i int32) (arrayInt64 []int64) {
	return ConvertArrayInt32toArrayInt64([]int32{i})
}

func NewContextWithMarker(ctx context.Context, key string, value string) context.Context {
	if key[0:1] != "_" {
		key = fmt.Sprint("_", key)
	}

	// return context.WithValue(ctx, key, logger.With(zap.String(key, strings.ToLower(value)))) // nolint
	return logger.ToContext(ctx, logger.Logger().With(zap.String(key, strings.ToLower(value))))
}

//	PercentageRoundedToAWhole Функция возвращает число являющееся процентом(percentageOfTarget) от target,
//
// всегда округленное в большую степень до целого числа
func PercentageRoundedToAWhole(percentageOfTarget float64, target float64) float64 {
	return math.Ceil(target / 100 * percentageOfTarget)
}

//	WhatPercentageRoundedToWhole Функция возвращает какой процент rps представляет от target,
//
// всегда округленное в большую степень до целого числа
func WhatPercentageRoundedToWhole(rps float64, target float64) float64 {
	return math.Ceil(rps / target * 100)
}

// DebugTimer Использовать только следующим образом:
// defer undecided.DebugTimer("lalala")()
// в конце функции в которой был вызван выведется затраченное время
func DebugTimer(ctx context.Context, name string) func() {
	start := time.Now()

	return func() {
		logger.Debugf(ctx, "Timer %s took %v", name, time.Since(start))
	}
}

// WarnTimer Использовать следующим образом:
// defer undecided.WarnTimer("lalala")()
// в конце функции в которой был вызван выведется затраченное время
func WarnTimer(ctx context.Context, name string) func() {
	start := time.Now()

	return func() {
		logger.Warnf(ctx, "Timer %s took %v", name, time.Since(start))
	}
}

// InfoTimer Использовать следующим образом:
// defer undecided.InfoTimer("lalala")()
// в конце функции в которой был вызван выведется затраченное время
func InfoTimer(ctx context.Context, name string) func() {
	start := time.Now()

	return func() {
		logger.Infof(ctx, "Timer %s took %v", name, time.Since(start))
	}
}

// RunLink Возвращает URl на РанID
func RunLink(ctx context.Context, runID int32) string {
	return fmt.Sprintf("%s/run/%d", BaseLink(ctx), runID)
}

// ScenarioLink Возвращает URl на сценарий
func ScenarioLink(ctx context.Context, projectID int32, scenarioID int32) string {
	if projectID < 0 {
		projectID = 0
	}
	return fmt.Sprintf("%s/project/%d/scenario/%d", BaseLink(ctx), projectID, scenarioID)
}

// ProjectLink Возвращает URl на проект
func ProjectLink(ctx context.Context, projectID int32) string {
	return fmt.Sprintf("%s/project/%d", BaseLink(ctx), projectID)
}

func BaseLink(ctx context.Context) string {
	env := config.Get(ctx).ENV
	switch env {
	case config.EnvInfra, config.EnvProd, config.EnvDev, config.EnvLocal:
		return "https://localhost:8080"
	}
	return ""
}

func BinarySearchInt64(arr []int64, target int64) int {
	left := 0
	right := len(arr) - 1

	for left <= right {
		mid := left + (right-left)/2

		if arr[mid] == target {
			return mid
		}

		if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return -1 // возвращаем -1, если элемент не найден
}
