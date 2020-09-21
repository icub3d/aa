package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrKeyNotFound     = errors.New("key not found")
	ErrUnsupportedType = errors.New("unsupported type (valid: int, bool, string)")
)

type Config struct {
	Environments map[string]Environment `yaml:"environments,omitempty"`
	Requests     map[string]Request     `yaml:"requests,omitempty"`
	Responses    map[string]Response    `yaml:"responses,omitempty"`
	Preferences  map[string]string      `yaml:"preferencesomitempty"`
}

func NewConfig(orgPath string) (*Config, error) {
	c := &Config{
		Environments: make(map[string]Environment),
		Requests:     make(map[string]Request),
		Responses:    make(map[string]Response),
		Preferences:  make(map[string]string),
	}
	err := filepath.Walk(orgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			nc := &Config{}
			buf, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			err = yaml.Unmarshal(buf, nc)
			if err != nil {
				return err
			}

			// The path should exclude the original path name and have
			// not trailing or leading slashes.
			path = strings.TrimPrefix(path, orgPath)
			path = filepath.Dir(path)
			path = strings.Trim(path, "/.")
			c.Merge(nc, path)
		}
		return nil
	})
	return c, err
}

func (c *Config) Merge(nc *Config, path string) {
	// Only include the trailing slash if we have a value.
	prefix := path
	if len(path) > 0 {
		prefix = path + "/"
	}

	// Environment and preferences are global, so shouldn't be affected by prefixes.
	for k, v := range nc.Environments {
		c.Environments[k] = v
	}

	for k, v := range nc.Preferences {
		c.Preferences[k] = v
	}

	for k, v := range nc.Requests {
		c.Requests[prefix+k] = v
	}

	for k, v := range nc.Responses {
		c.Responses[prefix+k] = v
	}
}
