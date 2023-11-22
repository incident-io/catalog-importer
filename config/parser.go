package config

import (
	"encoding/json"
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

	return parse(data)
}

func parse(data []byte) (*Config, error) {
	// Translate from JSON/YAML to a JSON string (normalise things so we can use the default
	// Go JSON parser).
	{
		var cfg map[string]any
		err := yaml.Unmarshal(data, &cfg)
		if err != nil {
			return nil, errors.Wrap(err, "parsing yaml/json")
		}

		data, err = json.Marshal(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "parsing yaml/json")
		}
	}

	d := json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()

	var cfg Config
	if err := d.Decode(&cfg); err != nil {
		return nil, errors.Wrap(err, "parsing config")
	}

	return &cfg, nil
}
