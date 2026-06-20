package analyze_test

import (
	"testing"

	"github.com/jkroepke/helm-release-size-analyzer/internal/analyze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTreeValidatedMeasuresNestedValues(t *testing.T) {
	t.Parallel()

	input := []byte("{\n  \"object\": {\"name\":\"value\", \"items\":[true, null]},\n  \"count\": 3\n}")

	tree, err := analyze.BuildTreeValidated(input)
	require.NoError(t, err)

	assert.Equal(t, "root", tree.Root.Name)
	assert.Equal(t, "object", tree.Root.Kind)
	assert.Equal(t, len(input), tree.Root.Bytes)
	require.Len(t, tree.Root.Children, 2)
	assert.Equal(t, "object", tree.Root.Children[0].Name)
	assert.Equal(t, len("\n  \"object\": {\"name\":\"value\", \"items\":[true, null]},\n  "), tree.Root.Children[0].Bytes)
	require.Len(t, tree.Root.Children[0].Children, 2)
	assert.Equal(t, "value", tree.Root.Children[0].Children[0].Preview)
	assert.Equal(t, "array", tree.Root.Children[0].Children[1].Kind)
	require.Len(t, tree.Root.Children[0].Children[1].Children, 2)
	assert.Equal(t, len("true, "), tree.Root.Children[0].Children[1].Children[0].Bytes)
	assert.Equal(t, "null", tree.Root.Children[0].Children[1].Children[1].Kind)
	assert.Equal(t, "3", tree.Root.Children[1].Preview)
}

func TestBuildTreeValidatedRejectsNonObject(t *testing.T) {
	t.Parallel()

	_, err := analyze.BuildTreeValidated([]byte(`["release"]`))
	require.Error(t, err)
}
