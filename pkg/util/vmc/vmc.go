package vmc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

type RegexError struct {
	Message string
	Code    string
}

func (e *RegexError) Error() string {
	return e.Message
}

const (
	ErrStatus = "failed"
)

func NewRegexError(message string, code string) *RegexError {
	return &RegexError{Message: message, Code: code}
}

// GetMetricsRange запрос к API VictoriaMetrics для получения метрик с использованием query_range
func GetMetricsRange(
	ctx context.Context,
	baseURL string, match string,
	start string,
	end string,
	stepSec string,
) (*APIResponse, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base url is not set")
	}

	if stepSec == "" {
		cfg := config.Get(ctx)
		stepSec = cfg.VMCStepSec
	}

	logger.Infof(ctx, "baseUrl: %v", baseURL)
	logger.Infof(ctx, "start: %v", start)
	logger.Infof(ctx, "end: %v", end)
	logger.Infof(ctx, "stepSec: %v", stepSec)
	logger.Infof(ctx, "query: %v", match)

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	u = u.JoinPath("api", "v1", "query_range")

	logger.Infof(ctx, "url.Parse 1: %v", u.String())

	q := u.Query()
	q.Add("start", start)
	q.Add("end", end)
	q.Add("step", stepSec)
	q.Add("query", match)

	u.RawQuery = q.Encode()
	logger.Infof(ctx, "url.Parse 2: %v", u.String())

	client := http.Client{}
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var apiResponse APIResponse
		if err = json.Unmarshal(body, &apiResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return &apiResponse, NewRegexError("Can not parse expression", ErrStatus)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResponse APIResponse
	if err = json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &apiResponse, nil
}

type APIResponse struct {
	Status    string `json:"status"`
	IsPartial bool   `json:"isPartial"`
	Data      Data   `json:"data"`
	Stats     Stats  `json:"stats"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

type Stats struct {
	SeriesFetched     string `json:"seriesFetched"`
	ExecutionTimeMsec int    `json:"executionTimeMsec"`
}
