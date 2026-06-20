package report_test

import (
	"bytes"
	"testing"

	"github.com/jkroepke/helm-release-size-analyser/internal/analyse"
	"github.com/jkroepke/helm-release-size-analyser/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTable(t *testing.T) {
	t.Parallel()

	input := analyse.Report{
		TotalBytes: 42,
		Properties: []analyse.Property{
			{Name: "name", Bytes: 17},
			{Name: "manifest", Bytes: 23},
		},
	}

	var output bytes.Buffer

	err := report.Write(&output, "table", input)
	require.NoError(t, err)

	want := "PROPERTY  BYTES\nTOTAL     42\nname      17\nmanifest  23\n"
	assert.Equal(t, want, output.String())
}

func TestWriteRejectsYAML(t *testing.T) {
	t.Parallel()

	err := report.Write(&bytes.Buffer{}, "yaml", analyse.Report{})
	require.Error(t, err)
}
