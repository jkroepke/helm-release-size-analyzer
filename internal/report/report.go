package report

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/jkroepke/helm-release-size-analyzer/internal/analyze"
)

// Write renders a report in the requested output format.
func Write(out io.Writer, format string, report analyze.Report) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(report)
		if err != nil {
			return fmt.Errorf("encode JSON report: %w", err)
		}

		return nil
	case "table":
		return writeTable(out, report)
	default:
		return fmt.Errorf("%w: %q", errUnsupportedFormat, format)
	}
}

// writeTable renders a human-readable size table.
func writeTable(out io.Writer, report analyze.Report) error {
	writer := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)

	_, err := fmt.Fprintln(writer, "PROPERTY\tSIZE")
	if err != nil {
		return fmt.Errorf("write table header: %w", err)
	}

	_, err = fmt.Fprintf(writer, "TOTAL\t%s\n", humanSize(report.TotalBytes))
	if err != nil {
		return fmt.Errorf("write total size: %w", err)
	}

	for _, property := range report.Properties {
		_, err = fmt.Fprintf(writer, "%s\t%s\n", property.Name, humanSize(property.Bytes))
		if err != nil {
			return fmt.Errorf("write property %q: %w", property.Name, err)
		}
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("flush table report: %w", err)
	}

	return nil
}

// humanSize formats a byte count in bytes or kibibyte-based kilobytes.
func humanSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%.2f B", float64(bytes))
	}

	return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
}
