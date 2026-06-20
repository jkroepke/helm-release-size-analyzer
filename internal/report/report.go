package report

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/jkroepke/helm-release-size-analyser/internal/analyse"
)

func Write(out io.Writer, format string, report analyse.Report) error {
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

func writeTable(out io.Writer, report analyse.Report) error {
	writer := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)

	_, err := fmt.Fprintln(writer, "PROPERTY\tBYTES")
	if err != nil {
		return fmt.Errorf("write table header: %w", err)
	}

	_, err = fmt.Fprintf(writer, "TOTAL\t%d\n", report.TotalBytes)
	if err != nil {
		return fmt.Errorf("write total size: %w", err)
	}

	for _, property := range report.Properties {
		_, err = fmt.Fprintf(writer, "%s\t%d\n", property.Name, property.Bytes)
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
