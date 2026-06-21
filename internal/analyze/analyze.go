package analyze

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
	Properties      []Property `json:"properties"`
	TotalBytes      int        `json:"total_bytes"`
	CompressedBytes int        `json:"compressed_bytes"`
}

type measuredProperty struct {
	Property

	valueStart int
	valueEnd   int
}

// Build measures the exact decoded JSON written to the Helm release Secret.
// Property sizes retain the original encoding and include delimiters and
// whitespace between top-level properties and direct children of chart.
func Build(releaseJSON []byte) (Report, error) {
	if !json.Valid(releaseJSON) {
		return Report{}, errInvalidPayload
	}

	return BuildValidated(releaseJSON)
}

// BuildValidated measures release JSON that the caller has already validated.
// Callers must not use this function with untrusted or unvalidated input.
func BuildValidated(releaseJSON []byte) (Report, error) {
	properties, err := measureProperties(releaseJSON)
	if err != nil {
		return Report{}, err
	}

	return Report{
		TotalBytes: len(releaseJSON),
		Properties: properties,
	}, nil
}

// measureProperties measures top-level properties and direct children of chart.
func measureProperties(data []byte) ([]Property, error) {
	return measureObjectProperties(data, "", true)
}

// measureObjectProperties measures each property in an encoded JSON object.
func measureObjectProperties(data []byte, prefix string, includeChartChildren bool) ([]Property, error) {
	measuredProperties, err := measureObjectSpans(data)
	if err != nil {
		return nil, err
	}

	properties := make([]Property, 0)

	for _, measured := range measuredProperties {
		measured.Name = prefix + measured.Name
		properties = append(properties, measured.Property)

		children, childErr := measureChartProperties(data, measured, includeChartChildren)
		if childErr != nil {
			return nil, childErr
		}

		properties = append(properties, children...)
	}

	return properties, nil
}

// measureChartProperties measures direct children when measured is the top-level chart object.
func measureChartProperties(
	data []byte,
	measured measuredProperty,
	enabled bool,
) ([]Property, error) {
	if !enabled || measured.Name != "chart" || data[measured.valueStart] != '{' {
		return nil, nil
	}

	properties, err := measureObjectProperties(
		data[measured.valueStart:measured.valueEnd],
		"chart.",
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("measure chart properties: %w", err)
	}

	return properties, nil
}

// measureObjectSpans locates every property span in an encoded JSON object.
func measureObjectSpans(data []byte) ([]measuredProperty, error) {
	cursor := skipWhitespace(data, 0)
	if cursor >= len(data) || data[cursor] != '{' {
		return nil, errTopLevelObject
	}

	cursor++
	properties := make([]measuredProperty, 0)

	for {
		measured, next, done, err := measureProperty(data, cursor)
		if err != nil {
			return nil, err
		}

		if measured.Bytes > 0 {
			properties = append(properties, measured)
		}

		if done {
			return properties, nil
		}

		cursor = next
	}
}

// measureProperty measures one encoded property and locates the following property.
func measureProperty(data []byte, propertyStart int) (measuredProperty, int, bool, error) {
	keyStart := skipWhitespace(data, propertyStart)
	if keyStart >= len(data) {
		return measuredProperty{}, 0, false, errObjectTerminated
	}

	if data[keyStart] == '}' {
		return measuredProperty{}, keyStart, true, nil
	}

	name, cursor, err := decodePropertyName(data, keyStart)
	if err != nil {
		return measuredProperty{}, 0, false, err
	}

	cursor = skipWhitespace(data, cursor)
	if cursor >= len(data) || data[cursor] != ':' {
		return measuredProperty{}, 0, false, fmt.Errorf("%w: %q", errValueSeparator, name)
	}

	valueStart := skipWhitespace(data, cursor+1)
	valueDecoder := json.NewDecoder(bytes.NewReader(data[valueStart:]))

	var value json.RawMessage

	err = valueDecoder.Decode(&value)
	if err != nil {
		return measuredProperty{}, 0, false, fmt.Errorf("decode top-level property %q: %w", name, err)
	}

	valueEnd := valueStart + int(valueDecoder.InputOffset())
	cursor = skipWhitespace(data, valueEnd)

	return finishProperty(data, propertyStart, cursor, valueStart, valueEnd, name)
}

// decodePropertyName decodes a JSON object key and returns the offset after it.
func decodePropertyName(data []byte, keyStart int) (string, int, error) {
	var name string

	decoder := json.NewDecoder(bytes.NewReader(data[keyStart:]))

	err := decoder.Decode(&name)
	if err != nil {
		return "", 0, fmt.Errorf("decode top-level property name: %w", err)
	}

	return name, keyStart + int(decoder.InputOffset()), nil
}

// finishProperty accounts for a property's delimiter and determines whether it ends the object.
func finishProperty(
	data []byte,
	propertyStart, cursor, valueStart, valueEnd int,
	name string,
) (measuredProperty, int, bool, error) {
	if cursor >= len(data) {
		return measuredProperty{}, 0, false, fmt.Errorf("%w: %q", errClosingBrace, name)
	}

	property := measuredProperty{
		Property:   Property{Name: name},
		valueStart: valueStart,
		valueEnd:   valueEnd,
	}

	switch data[cursor] {
	case ',':
		cursor = skipWhitespace(data, cursor+1)
		property.Bytes = cursor - propertyStart

		return property, cursor, false, nil
	case '}':
		// The closing brace belongs to the total JSON size, not a property.
		property.Bytes = cursor - propertyStart

		return property, cursor, true, nil
	default:
		return measuredProperty{}, 0, false, fmt.Errorf("%w: %q", errPropertyEnd, name)
	}
}

// skipWhitespace returns the first offset that does not contain JSON whitespace.
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
