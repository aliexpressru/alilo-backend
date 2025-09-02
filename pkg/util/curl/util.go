package curl

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/mattn/go-shellwords"
)

// var logger = zap.S()

type Headers map[string]string

type QueryParams struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Request struct {
	Method      string        `json:"method"`
	URL         string        `json:"url"`
	QueryParams []QueryParams `json:"query_params"`
	Headers     Headers       `json:"headers"`
	Body        string        `json:"body"`
}

const user = "user"
const userAgent = "user-agent"
const header = "header"
const data = "data"
const method = "method"
const cookie = "cookie"
const head = "HEAD"

func (r *Request) ToJSON(format bool) string {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)

	if format {
		encoder.SetIndent("", "  ")
	}

	_ = encoder.Encode(r)

	return buffer.String()
}

func Parse(ctx context.Context, curl string) (*Request, bool) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "ParseCUrl failed: '%+v'", err)
		}
	}()

	if strings.Index(curl, "curl ") != 0 {
		return nil, false
	}

	args, err := shellwords.Parse(curl)
	if err != nil {
		return nil, false
	}

	args = rewrite(args)
	request := &Request{
		Method:      "GET",
		Headers:     Headers{},
		QueryParams: []QueryParams{},
	}
	state := ""

	for _, arg := range args {
		switch true {
		case isURL(arg):
			if strings.Contains(arg, "?") {
				url := strings.Split(arg, "?")
				request.URL = url[0]

				if len(url) > 1 {
					if strings.Contains(url[1], "&") {
						queryParams := strings.Split(url[1], "&")
						for _, queryParam := range queryParams {
							if strings.Contains(queryParam, "=") {
								param := strings.Split(queryParam, "=")
								request.QueryParams = append(request.QueryParams,
									QueryParams{Key: param[0], Value: param[1]})
							} else {
								request.QueryParams = append(request.QueryParams, QueryParams{Key: queryParam})
							}
						}
					} else {
						if strings.Contains(url[1], "=") {
							param := strings.Split(url[1], "=")
							request.QueryParams = append(request.QueryParams,
								QueryParams{Key: param[0], Value: param[1]})
						} else {
							request.QueryParams = append(request.QueryParams, QueryParams{Key: url[1]})
						}
					}
				}
			} else {
				request.URL = arg
			}

		case arg == "-A" || arg == "--user-agent":
			state = userAgent

		case arg == "-H" || arg == "--header":
			state = header

		case arg == "-d" || arg == "--data" || arg == "--data-ascii" || arg == "--data-raw":
			state = data

		case arg == "-u" || arg == "--user":
			state = user

		case arg == "-I" || arg == "--head":
			request.Method = head

		case arg == "-X" || arg == "--request":
			state = method

		case arg == "-b" || arg == "--cookie":
			state = cookie
		case len(arg) > 0:
			switch state {
			case header:
				fields := parseField(arg)
				request.Headers[fields[0]] = strings.TrimSpace(fields[1])
				state = ""
			case userAgent:
				request.Headers["User-Agent"] = arg
				state = ""
			case data:
				if request.Method == "GET" || request.Method == head {
					request.Method = "POST"
				}

				if !hasContentType(*request) {
					request.Headers["Content-Type"] = "application/x-www-form-urlencoded"
				}

				if len(request.Body) == 0 {
					request.Body = arg
				} else {
					request.Body = request.Body + "&" + arg
				}

				state = ""
			case user:
				request.Headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(arg))
				state = ""
			case method:
				request.Method = arg
				state = ""
			case cookie:
				request.Headers["Cookie"] = arg
				state = ""
			default:
				fmt.Println("default: ", state)
			}
		}

	}

	// format json body
	if value, ok := request.Headers["Content-Type"]; ok && value == "application/json" {
		decoder := json.NewDecoder(strings.NewReader(request.Body))
		jsonData := make(map[string]interface{})

		if err = decoder.Decode(&jsonData); err == nil {
			buffer := &bytes.Buffer{}
			encoder := json.NewEncoder(buffer)
			encoder.SetEscapeHTML(false)

			if err = encoder.Encode(jsonData); err == nil {
				request.Body = strings.ReplaceAll(buffer.String(), "\n", "")
			}
		}
	}

	return request, true
}

func rewrite(args []string) []string {
	res := make([]string, 0)

	for _, arg := range args {
		arg = strings.TrimSpace(arg)

		if arg == "\n" {
			continue
		}

		if strings.Contains(arg, "\n") {
			arg = strings.ReplaceAll(arg, "\n", "")
		}

		// split request method
		if strings.Index(arg, "-X") == 0 {
			res = append(res, arg[0:2])
			res = append(res, arg[2:])
		} else {
			res = append(res, arg)
		}
	}

	return res
}

func isURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func parseField(arg string) []string {
	index := strings.Index(arg, ":")
	if index > 0 {
		return []string{arg[0:index], arg[index+2:]}
	}

	return []string{arg, ""}
}

func hasContentType(request Request) bool {
	if _, ok := request.Headers["Content-Type"]; ok {
		return true
	}

	return false
}
