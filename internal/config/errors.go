package config

import "errors"

var (
	errNamespace = errors.New("namespace must not be empty")
	errOutput    = errors.New("output must be one of table, json, or web")
	errLogLevel  = errors.New("log-level must be one of debug, info, warn, or error")
	errLogFormat = errors.New("log-format must be one of text or json")
)
