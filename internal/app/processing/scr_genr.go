package processing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/internal/app/processing/upload"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	file2 "github.com/aliexpressru/alilo-backend/pkg/util/file"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func getStages(ctx context.Context, goalTarget int, steps int, duration string) []map[string]interface{} {
	var myStages []map[string]interface{}

	stepsCount := int(math.Max(1, float64(steps)))
	stepTarget := int(math.Max(1, float64(goalTarget)/float64(stepsCount)))
	stepTime := divideTimeUnit(ctx, duration, stepsCount)

	for i := 1; i < stepsCount; i++ {
		myStages = append(myStages, map[string]interface{}{"target": i * stepTarget, "duration": "0s"})
		myStages = append(myStages, map[string]interface{}{"target": i * stepTarget, "duration": stepTime})
	}

	myStages = append(myStages, map[string]interface{}{"target": goalTarget, "duration": "0s"})
	myStages = append(myStages, map[string]interface{}{"target": goalTarget, "duration": stepTime})

	return myStages
}

func divideTimeUnit(ctx context.Context, timeUnit string, divider int) string {
	re := regexp.MustCompile(`(\d+)([s,mhd])`)
	myArray := re.FindStringSubmatch(timeUnit)
	timePostfix := myArray[2]
	t, _ := strconv.Atoi(myArray[1])

	logger.Infof(ctx, "divideTimeUnit initial: %v %v", timeUnit, divider)

	switch timePostfix {
	case "s":
		t = 1
	case "m":
		timePostfix = "s"
		t = int(math.Floor(float64(t) * 60))
	case "h":
		timePostfix = "s"
		t = int(math.Floor(float64(t) * 60 * 60))
	case "d":
		timePostfix = "m"
		t = int(math.Floor(float64(t) * 24 * 60))
	}

	t = t / divider
	logger.Infof(
		ctx,
		"divideTimeUnit resultant:%v timePostfix:%v, timeUnit:%v, divider:%v}",
		t,
		timePostfix,
		timeUnit,
		divider,
	)

	return fmt.Sprintf("%v%v", t, timePostfix)
}

// SimpleScriptGenerate Генерация скрипта.js для исполнения на агенте. Скрипт сохраняется в minio(s3)
func SimpleScriptGenerate(ctx context.Context, simpleScript *pb.SimpleScript, db *data.Store) (er error) {
	templatePath := "./templateSimpleScript"
	if file2.IsExist(templatePath) {
		simpleScriptTemplate, err := template.ParseFiles(templatePath)
		if err != nil {
			logger.Errorf(ctx, "Script file parsing error: '%v'", err)

			return err
		}

		intRps, err := strconv.Atoi(simpleScript.Rps)
		if err != nil {
			err = errors.Wrapf(err, "Cannot convert RPS param to int(%v)", simpleScript.Rps)
			logger.Errorf(ctx, "SimpleScriptGenerate error: '%v'", err)

			return err
		}

		if intRps <= 0 {
			err = errors.New(fmt.Sprint("rps must be positive and greater then 0 (", intRps, ")"))
			logger.Errorf(ctx, "SimpleScriptGenerate error: '%v'", err)

			return err
		}

		intSteps, err := strconv.Atoi(simpleScript.Steps)
		if err != nil {
			err = errors.Wrapf(err, "Cannot convert Steps param to int(%v)", simpleScript.Steps)
			logger.Errorf(ctx, "SimpleScriptGenerate error: '%v'", err)

			return err
		}

		if intSteps <= 0 {
			err = errors.New(fmt.Sprint("steps must be positive and greater then 0(", intSteps, ")"))
			logger.Errorf(ctx, "SimpleScriptGenerate error: '%v'", err)

			return err
		}

		stageArray := getStages(ctx, intRps, intSteps, simpleScript.Duration)

		jsonData, err := json.Marshal(stageArray)
		if err != nil {
			err = errors.Wrapf(err, "getStages JSON marshal error")
			logger.Errorf(ctx, "SimpleScriptGenerate error: '%v'", err)

			return err
		}

		checkAndAppendDefaultScriptParams(ctx, simpleScript)

		templateWithStages := struct {
			*pb.SimpleScript
			Stages string
		}{
			simpleScript,
			string(jsonData),
		}

		var b bytes.Buffer

		err = simpleScriptTemplate.Execute(&b, &templateWithStages)
		if err != nil {
			err = errors.Wrapf(err, "Execute templateSimpleScript error")
			logger.Errorf(ctx, "SimpleScriptGenerate error: '%v'", err)

			return err
		}

		project, err := db.GetMProject(ctx, simpleScript.ProjectId)
		if err != nil {
			err = errors.Wrapf(err, "Error getting mProject")
			logger.Errorf(ctx, "SimpleScriptGenerate: '%v'", err)

			return err
		}

		scenario, err := db.GetMScenario(ctx, simpleScript.ScenarioId)
		if err != nil {
			err = errors.Wrapf(err, "Error getting mScenario")
			logger.Errorf(ctx, "SimpleScriptGenerate: '%v'", err)

			return err
		}

		logger.Infof(ctx, "Unloading the created script(%v)", b.Len())
		byteSlice := b.Bytes()

		file, err := upload.ScriptToUpload(ctx,
			simpleScript.Name, &byteSlice,
			config.Get(ctx).MinioBucket, simpleScript.Description, project.Title, scenario.Title,
			nil)
		if err != nil {
			err = errors.Wrapf(err, "Error simpleScript to upload")
			logger.Errorf(ctx, "SimpleScriptGenerate: '%v'", err)

			return err
		}

		simpleScript.ScriptFileUrl = file.S3Url
	} else {
		mess := fmt.Sprintf("\tTemplate '%v' if not exist ", templatePath)
		return errors.New(mess)
	}

	return nil
}

// fixme: нужно стандартные параметры где-то хранить, что бы иметь возможность их контролировать без необходимости редеплоя
func checkAndAppendDefaultScriptParams(ctx context.Context, simpleScript *pb.SimpleScript) {
	logger.Debugf(ctx, "check simpleScript: %v", simpleScript.Name)

	checkAndAppendDefaultHeaders(ctx, simpleScript)

	checkAndAppendDefaultQueryParams(ctx, simpleScript)
}

func checkAndAppendDefaultHeaders(ctx context.Context, simpleScript *pb.SimpleScript) {
	dHeaders := strings.Split(config.Get(ctx).DefaultHeaders, ",")
	if len(dHeaders) < 1 {
		logger.Infof(ctx, "incorrect default Headers %s", dHeaders)

		return
	}

	for _, dHeader := range dHeaders {
		dHeaderKV := strings.Split(dHeader, ":")
		if len(dHeaderKV) < 2 {
			logger.Infof(ctx, "incorrect header %s", dHeader)

			continue
		}
		if s, ok := simpleScript.Headers[dHeaderKV[0]]; ok {
			if s == "" {
				simpleScript.Headers[dHeaderKV[0]] = dHeaderKV[1]
			}

			continue
		}
		simpleScript.Headers[dHeaderKV[0]] = dHeaderKV[1]
	}
}

func checkAndAppendDefaultQueryParams(ctx context.Context, simpleScript *pb.SimpleScript) {
	dQueryParams := strings.Split(config.Get(ctx).DefaultQueryParams, ",")
	if len(dQueryParams) < 1 {
		logger.Infof(ctx, "incorrect default QueryParams %s", dQueryParams)

		return
	}
	for _, dQueryParam := range dQueryParams {
		dQueryParamKV := strings.Split(dQueryParam, ":")
		if len(dQueryParamKV) < 2 {
			logger.Infof(ctx, "incorrect header %s", dQueryParam)

			continue
		}
		sort.SliceStable(simpleScript.QueryParams, func(i, j int) bool {
			return simpleScript.QueryParams[i].Key < simpleScript.QueryParams[j].Key
		})
		if i := binarySearchQueryParams(simpleScript.QueryParams, dQueryParamKV[0]); i == -1 {
			qp := &pb.QueryParams{
				Key:   dQueryParamKV[0],
				Value: dQueryParamKV[1],
			}
			simpleScript.QueryParams = append(simpleScript.QueryParams, qp)
		}
	}
}

func binarySearchQueryParams(arr []*pb.QueryParams, queryParamKey string) int {
	left := 0
	right := len(arr) - 1

	for left <= right {
		mid := left + (right-left)/2

		if arr[mid].Key == queryParamKey {
			return mid
		}

		if arr[mid].Key < queryParamKey {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return -1 // возвращаем -1, если элемент не найден
}
