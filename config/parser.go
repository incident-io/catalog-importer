package config

import (
	"strings"

	"github.com/ghodss/yaml"
	"github.com/google/go-jsonnet"
	"github.com/pkg/errors"
)

func Parse(filename string, data []byte) (*Config, error) {
	// If we think this is jsonnet, try parsing it.
	if strings.HasSuffix(filename, ".jsonnet") {
		jsonString, err := jsonnet.MakeVM().EvaluateAnonymousSnippet(filename, string(data))
		if err != nil {
			return nil, errors.Wrap(err, "parsing jsonnet")
		}

		data = []byte(jsonString)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "parsing yaml")
	}

	return &cfg, nil
}
