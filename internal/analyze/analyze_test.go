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
