package analyse_test

import (
	"testing"

	"github.com/jkroepke/helm-release-size-analyser/internal/analyse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMeasuresExactJSON(t *testing.T) {
	t.Parallel()

	releaseJSON := []byte("{\n  \"name\": \"release\\\"name\", \n  \"config\": {\"message\":\"hello world\"}\n}")

	report, err := analyse.Build(releaseJSON)
	require.NoError(t, err)
	assert.Equal(t, len(releaseJSON), report.TotalBytes)

	want := []analyse.Property{
		{Name: "name", Bytes: len("\n  \"name\": \"release\\\"name\", \n  ")},
		{Name: "config", Bytes: len("\"config\": {\"message\":\"hello world\"}\n")},
	}

	assert.Equal(t, want, report.Properties)
}

func TestBuildRejectsNonObject(t *testing.T) {
	t.Parallel()

	_, err := analyse.Build([]byte(`["release"]`))
	require.Error(t, err)
}
