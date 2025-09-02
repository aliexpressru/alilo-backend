package util

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

// CheckingTagForPresenceInDB проверка на наличие активных агентов с указанным тегом
func CheckingTagForPresenceInDB(ctx context.Context, tag string, db *data.Store) (tg string, err error) {
	tags, err := db.GetAllTags(ctx)
	if err != nil {
		err = fmt.Errorf("GetAllTags error{%w}", err)
		logger.Error(ctx, "Errrrr: ", err)

		return "", err
	}

	var contains bool

	for _, t := range tags {
		if tag == t {
			contains = true

			break
		}
	}

	if !contains {
		logger.Warnf(ctx, "invalid tag{%v}", tag)
		tag = ""
	}
	// Костыль для корректной работы с тегами в разных сегментах сети
	// На деве, только дев генераторы.
	// А в инфре разные генераторы(стейдж, дев, прод...),
	// и по умолчанию будет всегда использоваться только прод тег для запуска скриптов
	cfg := config.Get(ctx)
	if tag == "" && cfg.ENV == config.EnvInfra {
		tag = cfg.DefaultTag
	}

	return tag, nil
}
