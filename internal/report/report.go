package report

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"helm-release-size-analyser/internal/analyse"
	"sigs.k8s.io/yaml"
)

func Write(out io.Writer, format string, report analyse.Report) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	case "yaml":
		data, err := yaml.Marshal(report)
		if err != nil {
			return fmt.Errorf("encode YAML report: %w", err)
		}
		_, err = out.Write(data)
		return err
	case "table":
		return writeTable(out, report)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeTable(out io.Writer, report analyse.Report) error {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	rows := [][2]any{
		{"RELEASE", report.ReleaseName},
		{"NAMESPACE", report.Namespace},
		{"REVISION", report.Revision},
		{"SECRET", report.SecretName},
		{"STATUS", report.Status},
		{"LIMIT BYTES", report.LimitBytes},
		{"HELM PAYLOAD BYTES", report.Metrics.HelmStoragePayloadBytes},
		{"SECRET DATA BYTES", report.Metrics.SecretDataBytes},
		{"SERIALIZED SECRET BYTES", report.Metrics.SerializedSecretBytes},
		{"MANIFEST BYTES", report.Metrics.ManifestBytes},
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(w, "%v\t%v\n", row[0], row[1]); err != nil {
			return err
		}
	}
	return w.Flush()
}
