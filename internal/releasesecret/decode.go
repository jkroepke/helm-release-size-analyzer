package releasesecret

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
)

const gzipHeader = "\x1f\x8b\x08"

// DecodeJSON returns the exact JSON bytes stored in a Helm release Secret,
// after removing Helm's base64 and gzip storage encoding.
func DecodeJSON(secret *corev1.Secret) ([]byte, error) {
	if secret == nil {
		return nil, errNilSecret
	}

	payload, ok := secret.Data["release"]
	if !ok || len(payload) == 0 {
		return nil, fmt.Errorf("%w: %q", errMissingPayload, secret.Name)
	}

	decoded, err := base64.StdEncoding.AppendDecode(nil, payload)
	if err != nil {
		return nil, fmt.Errorf("decode Helm release payload: %w", err)
	}

	if !bytes.HasPrefix(decoded, []byte(gzipHeader)) {
		return nil, fmt.Errorf("%w: %q", errNotCompressed, secret.Name)
	}

	reader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("open Helm release payload: %w", err)
	}

	decoded, err = io.ReadAll(reader)
	closeErr := reader.Close()

	if err != nil {
		return nil, fmt.Errorf("decompress Helm release payload: %w", err)
	}

	if closeErr != nil {
		return nil, fmt.Errorf("close Helm release payload: %w", closeErr)
	}

	if !json.Valid(decoded) {
		return nil, fmt.Errorf("%w: %q", errInvalidJSON, secret.Name)
	}

	return decoded, nil
}
