package config

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

const DefaultLimitBytes = 1024 * 1024

type Config struct {
	ReleaseName string   `mapstructure:"release-name"`
	Namespace   string   `mapstructure:"namespace"`
	ValueFiles  []string `mapstructure:"values"`
	SetValues   []string `mapstructure:"set"`
	SetStrings  []string `mapstructure:"set-string"`
	SetFiles    []string `mapstructure:"set-file"`
	IncludeCRDs bool     `mapstructure:"include-crds"`
	LimitBytes  int      `mapstructure:"limit-bytes"`
	FailOn      string   `mapstructure:"fail-on"`
	Output      string   `mapstructure:"output"`
	LogLevel    string   `mapstructure:"log-level"`
	LogFormat   string   `mapstructure:"log-format"`
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Namespace) == "" {
		return fmt.Errorf("namespace must not be empty")
	}
	if c.LimitBytes <= 0 {
		return fmt.Errorf("limit-bytes must be greater than zero")
	}
	if !oneOf(c.FailOn, "never", "warning", "error") {
		return fmt.Errorf("fail-on must be one of never, warning, or error")
	}
	if !oneOf(c.Output, "table", "json", "yaml") {
		return fmt.Errorf("output must be one of table, json, or yaml")
	}
	if !oneOf(c.LogLevel, "debug", "info", "warn", "error") {
		return fmt.Errorf("log-level must be one of debug, info, warn, or error")
	}
	if !oneOf(c.LogFormat, "text", "json") {
		return fmt.Errorf("log-format must be one of text or json")
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	return slices.Contains(allowed, value)
}
