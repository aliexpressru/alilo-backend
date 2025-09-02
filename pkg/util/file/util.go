package file

import (
	"context"
	"os"
	"path/filepath"

	"github.com/aliexpressru/alilo-backend/pkg/util/common"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

// var logger = zap.S()

func IsExist(filePath string) (exist bool) {
	_, err := os.Stat(filePath)

	return err == nil
}

// ReadTheData - Получаем данные из файла
func ReadTheData(ctx context.Context, filePath string) (fileData *string) {
	file, err := ReadBytesFromFile(ctx, filePath)
	if err != nil {
		logger.Info(ctx, "err:\n", err)
	}

	return common.P(string(file))
}

// ReadBytesFromFile - Получаем байты из файла
func ReadBytesFromFile(ctx context.Context, filePath string) (fileData []byte, err error) {
	file, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		logger.Info(ctx, "ReadBytesFromFile err:\n", err)
	}

	return file, err
}
