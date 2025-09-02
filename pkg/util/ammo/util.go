package ammo

import (
	"context"
	"encoding/json"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"gopkg.in/yaml.v3"
)

type Params map[string]string
type Headers map[string]string
type Data string

type Ammo struct {
	Namespace string  `yaml:"namespace"`
	Method    string  `yaml:"method"`
	Params    Params  `yaml:"params,omitempty"`
	Headers   Headers `yaml:"headers,omitempty"`
	Data      Data    `yaml:"data,omitempty"`
}

func ValidateYaml(ctx context.Context, file []byte) (err error) {
	var ammoStore []Ammo

	err = yaml.Unmarshal(file, &ammoStore)
	if err != nil {
		logger.Errorf(ctx, "Unmarshal Yaml error: %v", err)
		return err
	}

	return nil
}

func ValidateJSON(ctx context.Context, file []byte) (err error) {
	var js json.RawMessage

	err = json.Unmarshal(file, &js)
	if err != nil {
		logger.Errorf(ctx, "Unmarshal Json error: %v", err)
		return err
	}

	return nil
}
