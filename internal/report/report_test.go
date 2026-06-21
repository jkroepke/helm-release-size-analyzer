package report_test

import (
	"bytes"
	"testing"

	"github.com/jkroepke/helm-release-size-analyzer/internal/analyze"
	"github.com/jkroepke/helm-release-size-analyzer/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTable(t *testing.T) {
	t.Parallel()

	input := analyze.Report{
		TotalBytes:      2048,
		CompressedBytes: 1024,
		Properties: []analyze.Property{
			{Name: "name", Bytes: 17},
			{Name: "manifest", Bytes: 1536},
		},
	}

	var output bytes.Buffer

	err := report.Write(&output, "table", input)
	require.NoError(t, err)

	want := "PROPERTY    SIZE\nTOTAL       2.00 KB\nCOMPRESSED  1.00 KB\nmanifest    1.50 KB\nname        17.00 B\n"
	assert.Equal(t, want, output.String())
	assert.Equal(t, "name", input.Properties[0].Name)
}

func TestWriteRejectsYAML(t *testing.T) {
	t.Parallel()

	err := report.Write(&bytes.Buffer{}, "yaml", analyze.Report{})
	require.Error(t, err)
}
