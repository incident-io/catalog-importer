package config

import (
	"context"
	"io/ioutil"
	"time"

	kitlog "github.com/go-kit/kit/log"
)

type Loader interface {
	Load(context.Context) (*Config, error)
}

type LoaderFunc func(context.Context) (*Config, error)

func (l LoaderFunc) Load(ctx context.Context) (*Config, error) {
	return l(ctx)
}

// FileLoader loads config from a filepath
type FileLoader string

func (l FileLoader) Load(context.Context) (*Config, error) {
	data, err := ioutil.ReadFile(string(l))
	if err != nil {
		return nil, err
	}

	return Parse(string(l), data)
}

// NewCachedLoader caches a loader to avoid repeated lookups.
func NewCachedLoader(logger kitlog.Logger, loader Loader, ttl time.Duration) Loader {
	return &cachedLoader{
		logger: logger,
		loader: loader,
		ttl:    ttl,
	}
}

type cachedLoader struct {
	logger      kitlog.Logger
	loader      Loader
	ttl         time.Duration
	cfg         *Config
	lastUpdated time.Time
}

func (c *cachedLoader) Load(ctx context.Context) (cfg *Config, err error) {
	if c.cfg == nil || time.Since(c.lastUpdated) > c.ttl {
		c.logger.Log("event", "loading_cofig", "msg", "cache expired, loading config")
		cfg, err := c.loader.Load(ctx)
		if err != nil {
			return nil, err
		}

		c.cfg = cfg
		c.lastUpdated = time.Now()
	}

	return c.cfg, nil
}
