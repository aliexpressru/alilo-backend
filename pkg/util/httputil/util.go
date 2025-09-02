package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	ContentType = "application/json"
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
)

// var logger = zap.S()

var (
	marshaller  = protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}
	unmarshaler = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}
	httpClient  = &http.Client{
		Timeout: 30 * time.Second,
	}
)

type ProxyCallConfig struct {
	URL          string
	Path         string
	RequestBody  protoreflect.ProtoMessage
	ResponseBody protoreflect.ProtoMessage
	Method       string
	Headers      []http.Header
}

func URL(scheme, host, path string) url.URL {
	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}

	return u
}

func GetWithHeaders(ctx context.Context, uri string, headers map[string]string) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf(ctx, "Recovered in GetWithHeaders(): %+v", r)
		}
	}()

	logger.Infof(ctx, "------- START http.Get: '%s'", uri)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		logger.Warn(ctx, "http.GetWithHeaders Error: ", err)
		return []byte{}, err
	}
	for key, val := range headers {
		req.Header.Add(key, val)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Warn(ctx, "http.GetWithHeaders Error: ", err)
		return []byte{}, err
	}
	defer func(resp *http.Response) {
		err = resp.Body.Close()
		if err != nil {
			logger.Warn(ctx, "http.GetWithHeaders Error: ", err)
		}
	}(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Warn(ctx, "Ошибка чтения строки из io.Reader")
		return []byte{}, err
	}

	return body, nil
}

func Get(ctx context.Context, uri string) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf(ctx, "Recovered in http.Get(): %+v", r)
		}
	}()

	logger.Infof(ctx, "------- START http.Get: '%s'", uri)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		logger.Warn(ctx, "http.Get Error: ", err)

		return []byte{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Warn(ctx, "http.Get Error: ", err)

		return []byte{}, err
	}
	defer func(resp *http.Response) {
		err = resp.Body.Close()
		if err != nil {
			logger.Warn(ctx, "http.Get Error: ", err)
		}
	}(resp)

	body := GetAndConvertReadToBytes(ctx, resp.Body)

	return body, nil
}

func Post(ctx context.Context,
	host string, contentType string, headers map[string]string, rqBody interface{}) (
	result []byte, err error) {
	defer func() {
		if er := recover(); er != nil {
			logger.Errorf(ctx, "Post failed: '%+v'", er)
		}
	}()

	defer undecided.WarnTimer(ctx, fmt.Sprintf("Post Request  %v", host))()

	bytesMarshal, err := json.Marshal(rqBody)
	if err != nil {
		logger.Errorf(ctx, "Post rq Marshal cfg.ENV, envRq: %v", err)

		return result, err
	}

	buffer := bytes.NewBuffer(bytesMarshal)
	logger.Infof(ctx, "Execution of POST request %v : %v", host, buffer.String())
	rq, err := http.NewRequest(http.MethodPost, host, buffer)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		rq.Header.Add(k, v)
	}
	rq.Header.Set("Content-Type", contentType)
	resp, err := httpClient.Do(rq)
	if err != nil {
		logger.Errorf(ctx, "PostExecution ERROR: %v", err)

		return result, err
	}

	defer func(resp *http.Response) {
		err2 := resp.Body.Close()
		if err2 != nil {
			logger.Errorf(ctx, "resp.Body.Close Error: %v", err2.Error())
		}
	}(resp)
	logger.Debugf(ctx, "Post '%v' Request: '%+v'", host, resp.Request)
	logger.Debugf(ctx, "Post '%v' Headers: '%+v'", host, resp.Header)
	logger.Infof(ctx, "Post '%v' Status: '%+v'", host, resp.Status)

	result, err = io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf(ctx, "Post ReadAll error: %v", err)

		return result, err
	}
	logger.Debugf(ctx, "Post resp.Body: '%s'", string(result))
	return result, err
}

func GetResponseCode(ctx context.Context, uri string, headers map[string]string) (int, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf(ctx, "Recovered in GetResponseCode(): %+v", r)
		}
	}()

	logger.Infof(ctx, "--- START http.Get: '%s'", uri)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		logger.Warn(ctx, "http.GetResponseCode NewRequest error: ", err)
		return -1, err
	}
	for key, val := range headers {
		req.Header.Add(key, val)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Warn(ctx, "http.GetResponseCode Error: ", err)
		return resp.StatusCode, err
	}
	defer func(resp *http.Response) {
		err = resp.Body.Close()
		if err != nil {
			logger.Warn(ctx, "http.GetResponseCode Close error: ", err)
		}
	}(resp)

	return resp.StatusCode, nil
}

func IsLocalhost(ctx context.Context, r *http.Request) bool {
	addr := r.RemoteAddr

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		logger.Warn(ctx, "Ошибка обработки адреса")
		return false
	}

	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// GetAndConvertReadToBytes Получаем байты из ио.ридера
func GetAndConvertReadToBytes(ctx context.Context, reader io.Reader) []byte {
	buf := new(bytes.Buffer)

	_, err := buf.ReadFrom(reader)
	if err != nil {
		logger.Warn(ctx, "Ошибка чтения строки из io.Reader")
	}

	return buf.Bytes()
}

func ProxyCall(ctx context.Context, config ProxyCallConfig) error {
	dstURL := fmt.Sprint(strings.Trim(config.URL, "/"), config.Path)
	requestBodyBytes, err := marshaller.Marshal(config.RequestBody)
	if err != nil {
		logger.Errorf(ctx, "proxyCall Marshal error: %v -> %v", dstURL, err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx, config.Method, dstURL, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		logger.Errorf(ctx, "newRequestWithContext error: %v -> %v", dstURL, err)

		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for _, h := range config.Headers {
		for key, values := range h {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Errorf(ctx, "HTTP RQ Do error: %v -> %v", config.URL, err)

		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		logger.Errorf(ctx, "HTTP RQ Do status code: %v -> %v", config.URL, resp.StatusCode)

		return errors.Errorf("request failed with status code %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf(ctx, "ReadAll Body error: %v -> %v", config.URL, err)

		return errors.Wrap(err, "read body failed")
	}
	err = unmarshaler.Unmarshal(b, config.ResponseBody)
	if err != nil {
		logger.Errorf(ctx, "Unmarshal responseBody error: %v -> %v", config.URL, err)

		return err
	}
	return nil
}
