package analyze_test

import (
	"testing"

	"github.com/jkroepke/helm-release-size-analyzer/internal/analyze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMeasuresExactJSON(t *testing.T) {
	t.Parallel()

	releaseJSON := []byte("{\n  \"name\": \"release\\\"name\", \n  \"config\": {\"message\":\"hello world\"}\n}")

	report, err := analyze.Build(releaseJSON)
	require.NoError(t, err)
	assert.Equal(t, len(releaseJSON), report.TotalBytes)

	want := []analyze.Property{
		{Name: "name", Bytes: len("\n  \"name\": \"release\\\"name\", \n  ")},
		{Name: "config", Bytes: len("\"config\": {\"message\":\"hello world\"}\n")},
	}

	assert.Equal(t, want, report.Properties)
}

func TestBuildValidatedMeasuresValidJSON(t *testing.T) {
	t.Parallel()

	releaseJSON := []byte(`{"name":"release"}`)

	report, err := analyze.BuildValidated(releaseJSON)
	require.NoError(t, err)
	assert.Equal(t, len(releaseJSON), report.TotalBytes)
	assert.Equal(t, []analyze.Property{{Name: "name", Bytes: len(`"name":"release"`)}}, report.Properties)
}

func TestBuildRejectsNonObject(t *testing.T) {
	t.Parallel()

	_, err := analyze.Build([]byte(`["release"]`))
	require.Error(t, err)
}

func TestBuildMeasuresChartProperties(t *testing.T) {
	t.Parallel()

	releaseJSON := []byte("{\"chart\": {\n  \"metadata\":{\"name\":\"example\"}, \n  \"values\":{\"message\":\"hello\"}\n},\"name\":\"release\"}")

	report, err := analyze.Build(releaseJSON)
	require.NoError(t, err)

	want := []analyze.Property{
		{Name: "chart", Bytes: len("\"chart\": {\n  \"metadata\":{\"name\":\"example\"}, \n  \"values\":{\"message\":\"hello\"}\n},")},
		{Name: "chart.metadata", Bytes: len("\n  \"metadata\":{\"name\":\"example\"}, \n  ")},
		{Name: "chart.values", Bytes: len("\"values\":{\"message\":\"hello\"}\n")},
		{Name: "name", Bytes: len("\"name\":\"release\"")},
	}

	assert.Equal(t, want, report.Properties)
}

func TestBuildMeasuresEscapedPropertyNamesAndValues(t *testing.T) {
	t.Parallel()

	releaseJSON := []byte("{\n\t\"escaped\\\"key\":\"comma, brace } bracket ] slash \\\\ null \\u0000\", \r\n" +
		"\"unicode\\u263a\":\"caf\xc3\xa9\",\"chart\":{\"nested\\\\key\":\"{\\\"still\\\":\\\"a string\\\"}\"}}")

	report, err := analyze.Build(releaseJSON)
	require.NoError(t, err)

	assert.Equal(t, len(releaseJSON), report.TotalBytes)
	assert.Equal(t, []analyze.Property{
		{Name: "escaped\"key", Bytes: len("\n\t\"escaped\\\"key\":\"comma, brace } bracket ] slash \\\\ null \\u0000\", \r\n")},
		{Name: "unicode\u263a", Bytes: len("\"unicode\\u263a\":\"caf\xc3\xa9\",")},
		{Name: "chart", Bytes: len("\"chart\":{\"nested\\\\key\":\"{\\\"still\\\":\\\"a string\\\"}\"}")},
		{Name: "chart.nested\\key", Bytes: len("\"nested\\\\key\":\"{\\\"still\\\":\\\"a string\\\"}\"")},
	}, report.Properties)
}

func TestBuildDecodesNullCharacterInPropertyName(t *testing.T) {
	t.Parallel()

	releaseJSON := []byte(`{"nul\u0000key":null}`)

	report, err := analyze.Build(releaseJSON)
	require.NoError(t, err)
	require.Len(t, report.Properties, 1)
	assert.Equal(t, "nul\x00key", report.Properties[0].Name)
	assert.Equal(t, len(`"nul\u0000key":null`), report.Properties[0].Bytes)
}

func TestBuildRejectsMalformedCharacters(t *testing.T) {
	t.Parallel()

	tests := map[string][]byte{
		"literal null byte":           []byte("{\"name\":\"bad\x00value\"}"),
		"invalid escape":              []byte(`{"name":"bad\qvalue"}`),
		"unterminated escaped string": []byte(`{"name":"bad\"}`),
		"trailing data":               []byte(`{"name":"release"}\x00`),
	}

	for name, releaseJSON := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := analyze.Build(releaseJSON)
			require.Error(t, err)
		})
	}
}
