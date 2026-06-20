package config

import (
	"slices"
	"strings"
)

type Config struct {
	ReleaseName string
	Namespace   string
	Output      string
	LogLevel    string
	LogFormat   string
	ValueFiles  []string
	SetValues   []string
	SetStrings  []string
	SetFiles    []string
	IncludeCRDs bool
}

// Validate checks that all configuration values satisfy the CLI contract.
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

// oneOf reports whether value is present in allowed.
func oneOf(value string, allowed ...string) bool {
	return slices.Contains(allowed, value)
}
