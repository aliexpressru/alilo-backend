package config

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func init() {
	println("Init config")

	defer func() {
		if err := recover(); err != nil {
			println(err)
			log.Fatalln(err)
		}
	}()
	// ctx := context.Background()

	ctx := logger.ToContext(
		context.Background(),
		logger.Logger().With(zap.String("_alilo", strings.ToLower("init_config"))),
	)
	err := loadLocalValuesYaml(ctx)
	if err != nil {
		println(err.Error())
		logger.Errorf(ctx, "Init config err: %v", err)

		return
	}

	println("Init config success")
}

func loadLocalValuesYaml(ctx context.Context) error {
	var localFile string

	for i, v := range os.Args {
		if v == "--local-config" && i+1 < len(os.Args) {
			localFile = os.Args[i+1]

			break
		}

		if strings.HasPrefix(v, "--local-config=") {
			parts := strings.SplitN(v, "=", 2)
			localFile = parts[1]

			break
		}
	}

	if localFile != "" {
		if err := loadEnvFromValuesFile(ctx, localFile); err != nil {
			return err
		}
	}

	return nil
}

func loadEnvFromValuesFile(ctx context.Context, filePath string) error {
	b, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return errors.Wrap(err, "read yaml file")
	}

	var config valuesYamlConfig
	if err := yaml.Unmarshal(b, &config); err != nil {
		return errors.Wrapf(err, "unmarshal %s", filePath)
	}

	for _, v := range config.Env {
		if err := os.Setenv(v.Name, v.Value); err != nil {
			return errors.Wrapf(err, "set env %s='%s'", v.Name, v.Value)
		}

		// logger.With(
		//	zap.String("config_file", file),
		// ).Debugf("set env %s='%s'", v.Name, v.Value)
		logger.Debugf(ctx, "set env %s='%s'", v.Name, v.Value)
	}

	return nil
}

type valuesYamlConfig struct {
	Env []struct {
		Name  string `yaml:"name"`
		Value string `yaml:"value"`
	} `yaml:"env"`
}
