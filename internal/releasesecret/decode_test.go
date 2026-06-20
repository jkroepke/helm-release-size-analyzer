package releasesecret_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"testing"

	"github.com/jkroepke/helm-release-size-analyzer/internal/releasesecret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	want := []byte(`{"name":"example","version":1}`)

	var compressed bytes.Buffer

	writer := gzip.NewWriter(&compressed)

	_, err := writer.Write(want)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	secret := &corev1.Secret{Data: map[string][]byte{
		"release": []byte(base64.StdEncoding.EncodeToString(compressed.Bytes())),
	}}

	got, err := releasesecret.DecodeJSON(secret)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDecodeJSONSupportsUncompressedPayload(t *testing.T) {
	t.Parallel()

	want := []byte(`{"name":"legacy"}`)
	secret := &corev1.Secret{Data: map[string][]byte{
		"release": []byte(base64.StdEncoding.EncodeToString(want)),
	}}

	got, err := releasesecret.DecodeJSON(secret)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDecodeJSONRejectsInvalidBase64(t *testing.T) {
	t.Parallel()

	secret := &corev1.Secret{Data: map[string][]byte{"release": []byte("not base64")}}

	_, err := releasesecret.DecodeJSON(secret)
	require.ErrorContains(t, err, "decode Helm release payload")
}
