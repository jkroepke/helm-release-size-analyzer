package analyse

import (
	"encoding/json"
	"fmt"

	"helm-release-size-analyser/internal/helminstall"
)

type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
)

type Metrics struct {
	HelmStoragePayloadBytes int `json:"helm_storage_payload_bytes" yaml:"helm_storage_payload_bytes"`
	SecretDataBytes         int `json:"secret_data_bytes" yaml:"secret_data_bytes"`
	SerializedSecretBytes   int `json:"serialized_secret_json_bytes" yaml:"serialized_secret_json_bytes"`
	ManifestBytes           int `json:"manifest_bytes" yaml:"manifest_bytes"`
	RecordedResourceBytes   int `json:"recorded_resource_bytes" yaml:"recorded_resource_bytes"`
}

type Report struct {
	ReleaseName string   `json:"release_name" yaml:"release_name"`
	Namespace   string   `json:"namespace" yaml:"namespace"`
	Revision    int      `json:"revision" yaml:"revision"`
	SecretName  string   `json:"secret_name" yaml:"secret_name"`
	Status      Status   `json:"status" yaml:"status"`
	LimitBytes  int      `json:"limit_bytes" yaml:"limit_bytes"`
	Metrics     Metrics  `json:"metrics" yaml:"metrics"`
	Warnings    []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

func Build(result helminstall.Result, limitBytes int) (Report, error) {
	serializedSecret, err := json.Marshal(result.Secret)
	if err != nil {
		return Report{}, fmt.Errorf("serialize release secret: %w", err)
	}

	secretDataBytes := 0
	for _, value := range result.Secret.Data {
		secretDataBytes += len(value)
	}

	payloadBytes := len(result.Secret.Data["release"])
	status := StatusOK
	if secretDataBytes > limitBytes {
		status = StatusError
	} else if secretDataBytes*100 >= limitBytes*80 {
		status = StatusWarning
	}

	warnings := []string{
		"Kubernetes API validation, defaulting, controllers, and admission are not simulated",
		"Helm hooks are disabled",
	}

	return Report{
		ReleaseName: result.Release.Name(),
		Namespace:   result.Release.Namespace(),
		Revision:    result.Release.Version(),
		SecretName:  result.Secret.Name,
		Status:      status,
		LimitBytes:  limitBytes,
		Metrics: Metrics{
			HelmStoragePayloadBytes: payloadBytes,
			SecretDataBytes:         secretDataBytes,
			SerializedSecretBytes:   len(serializedSecret),
			ManifestBytes:           len(result.Release.Manifest()),
			RecordedResourceBytes:   result.RecordedBytes,
		},
		Warnings: warnings,
	}, nil
}
