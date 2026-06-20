package analyze

import "errors"

var (
	errInvalidPayload   = errors.New("release payload is not valid JSON")
	errTopLevelObject   = errors.New("release JSON must be a top-level object")
	errObjectTerminated = errors.New("release JSON object is not terminated")
	errValueSeparator   = errors.New("top-level property has no value separator")
	errClosingBrace     = errors.New("top-level property is not followed by a closing brace")
	errPropertyEnd      = errors.New("top-level property is not followed by a comma or closing brace")
)
