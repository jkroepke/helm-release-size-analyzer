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

func TestDecodeJSONRejectsUncompressedPayload(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"name":"legacy"}`)
	secret := &corev1.Secret{Data: map[string][]byte{
		"release": []byte(base64.StdEncoding.EncodeToString(payload)),
	}}

	_, err := releasesecret.DecodeJSON(secret)
	require.ErrorContains(t, err, "release secret payload is not gzip-compressed")
}

func TestDecodeJSONRejectsInvalidBase64(t *testing.T) {
	t.Parallel()

	secret := &corev1.Secret{Data: map[string][]byte{"release": []byte("not base64")}}

	_, err := releasesecret.DecodeJSON(secret)
	require.ErrorContains(t, err, "decode Helm release payload")
}

func TestDecodeJSONRejectsCorruptSecrets(t *testing.T) {
	t.Parallel()

	validCompressed := gzipPayload(t, []byte(`{"name":"example"}`))
	corruptChecksum := append([]byte(nil), validCompressed...)
	corruptChecksum[len(corruptChecksum)-1] ^= 0xff

	tests := map[string]struct {
		secret  *corev1.Secret
		message string
	}{
		"nil secret": {
			message: "release secret is nil",
		},
		"nil data": {
			secret:  &corev1.Secret{},
			message: "release secret has no release payload",
		},
		"missing payload": {
			secret:  &corev1.Secret{Data: map[string][]byte{"other": []byte("value")}},
			message: "release secret has no release payload",
		},
		"empty payload": {
			secret:  &corev1.Secret{Data: map[string][]byte{"release": nil}},
			message: "release secret has no release payload",
		},
		"valid base64 with invalid JSON": {
			secret:  secretWithPayload(gzipPayload(t, []byte(`{"name":`))),
			message: "invalid release JSON payload",
		},
		"literal null byte in JSON": {
			secret:  secretWithPayload(gzipPayload(t, []byte("{\"name\":\"bad\x00value\"}"))),
			message: "invalid release JSON payload",
		},
		"truncated gzip header": {
			secret:  secretWithPayload([]byte{0x1f, 0x8b, 0x08}),
			message: "open Helm release payload",
		},
		"truncated gzip body": {
			secret:  secretWithPayload(validCompressed[:len(validCompressed)-4]),
			message: "decompress Helm release payload",
		},
		"corrupt gzip checksum": {
			secret:  secretWithPayload(corruptChecksum),
			message: "decompress Helm release payload",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := releasesecret.DecodeJSON(test.secret)
			require.ErrorContains(t, err, test.message)
		})
	}
}

func gzipPayload(t *testing.T, payload []byte) []byte {
	t.Helper()

	var compressed bytes.Buffer

	writer := gzip.NewWriter(&compressed)

	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return compressed.Bytes()
}

func secretWithPayload(payload []byte) *corev1.Secret {
	return &corev1.Secret{Data: map[string][]byte{
		"release": []byte(base64.StdEncoding.EncodeToString(payload)),
	}}
}
