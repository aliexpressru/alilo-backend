package processing

import (
	"context"
	"fmt"

	curlUtils "github.com/aliexpressru/alilo-backend/pkg/util/curl"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func ParseCUrl(ctx context.Context, curl string) (json, message string) {
	defer func() {
		if err := recover(); err != nil {
			message = fmt.Sprintf("recover. ParseCUrl failed: %+v", err)
			logger.Errorf(ctx, "ParseCUrl: '%+v'", message)
		}
	}()
	logger.Infof(ctx, "ParseCUrl. Send data: '%+v'", curl)

	parse, result := curlUtils.Parse(ctx, curl)
	if result {
		j := parse.ToJSON(true)
		logger.Infof(ctx, "Parse Curl to json: %+v", j)

		return j, message
	}

	return "", "Error parse"
}
