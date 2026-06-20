package analyse

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Property struct {
	Name  string `json:"name"`
	Bytes int    `json:"bytes"`
}

type Report struct {
	Properties []Property `json:"properties"`
	TotalBytes int        `json:"total_bytes"`
}

// Build measures the exact decoded JSON written to the Helm release Secret.
// Property sizes retain the original encoding and include delimiters and
// whitespace between top-level properties.
func Build(releaseJSON []byte) (Report, error) {
	if !json.Valid(releaseJSON) {
		return Report{}, errInvalidPayload
	}

	properties, err := measureProperties(releaseJSON)
	if err != nil {
		return Report{}, err
	}

	return Report{
		TotalBytes: len(releaseJSON),
		Properties: properties,
	}, nil
}

func measureProperties(data []byte) ([]Property, error) {
	cursor := skipWhitespace(data, 0)
	if cursor >= len(data) || data[cursor] != '{' {
		return nil, errTopLevelObject
	}

	cursor++
	properties := make([]Property, 0)

	for {
		property, next, done, err := measureProperty(data, cursor)
		if err != nil {
			return nil, err
		}

		if done {
			if property.Bytes > 0 {
				properties = append(properties, property)
			}

			return properties, nil
		}

		properties = append(properties, property)
		cursor = next
	}
}

func measureProperty(data []byte, propertyStart int) (Property, int, bool, error) {
	keyStart := skipWhitespace(data, propertyStart)
	if keyStart >= len(data) {
		return Property{}, 0, false, errObjectTerminated
	}

	if data[keyStart] == '}' {
		return Property{}, keyStart, true, nil
	}

	name, cursor, err := decodePropertyName(data, keyStart)
	if err != nil {
		return Property{}, 0, false, err
	}

	cursor = skipWhitespace(data, cursor)
	if cursor >= len(data) || data[cursor] != ':' {
		return Property{}, 0, false, fmt.Errorf("%w: %q", errValueSeparator, name)
	}

	valueStart := skipWhitespace(data, cursor+1)
	valueDecoder := json.NewDecoder(bytes.NewReader(data[valueStart:]))

	var value json.RawMessage

	err = valueDecoder.Decode(&value)
	if err != nil {
		return Property{}, 0, false, fmt.Errorf("decode top-level property %q: %w", name, err)
	}

	cursor = skipWhitespace(data, valueStart+int(valueDecoder.InputOffset()))

	return finishProperty(data, propertyStart, cursor, name)
}

func decodePropertyName(data []byte, keyStart int) (string, int, error) {
	var name string

	decoder := json.NewDecoder(bytes.NewReader(data[keyStart:]))

	err := decoder.Decode(&name)
	if err != nil {
		return "", 0, fmt.Errorf("decode top-level property name: %w", err)
	}

	return name, keyStart + int(decoder.InputOffset()), nil
}

func finishProperty(data []byte, propertyStart, cursor int, name string) (Property, int, bool, error) {
	if cursor >= len(data) {
		return Property{}, 0, false, fmt.Errorf("%w: %q", errClosingBrace, name)
	}

	switch data[cursor] {
	case ',':
		cursor = skipWhitespace(data, cursor+1)

		return Property{Name: name, Bytes: cursor - propertyStart}, cursor, false, nil
	case '}':
		// The closing brace belongs to the total JSON size, not a property.
		return Property{Name: name, Bytes: cursor - propertyStart}, cursor, true, nil
	default:
		return Property{}, 0, false, fmt.Errorf("%w: %q", errPropertyEnd, name)
	}
}

func skipWhitespace(data []byte, offset int) int {
	for offset < len(data) {
		switch data[offset] {
		case ' ', '\t', '\n', '\r':
			offset++
		default:
			return offset
		}
	}

	return offset
}
