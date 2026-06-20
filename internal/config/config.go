package config

import (
	"slices"
	"strings"
)

type Config struct {
	ReleaseName string   `mapstructure:"release-name"`
	Namespace   string   `mapstructure:"namespace"`
	Output      string   `mapstructure:"output"`
	LogLevel    string   `mapstructure:"log-level"`
	LogFormat   string   `mapstructure:"log-format"`
	ValueFiles  []string `mapstructure:"values"`
	SetValues   []string `mapstructure:"set"`
	SetStrings  []string `mapstructure:"set-string"`
	SetFiles    []string `mapstructure:"set-file"`
	IncludeCRDs bool     `mapstructure:"include-crds"`
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Namespace) == "" {
		return errNamespace
	}

	if !oneOf(c.Output, "table", "json") {
		return errOutput
	}

	if !oneOf(c.LogLevel, "debug", "info", "warn", "error") {
		return errLogLevel
	}

	if !oneOf(c.LogFormat, "text", "json") {
		return errLogFormat
	}

	return nil
}

func oneOf(value string, allowed ...string) bool {
	return slices.Contains(allowed, value)
}
