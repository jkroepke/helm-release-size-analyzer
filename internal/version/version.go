package version

import "fmt"

var (
	Version   = "dev"     //nolint:gochecknoglobals // Populated by GoReleaser through ldflags.
	Revision  = "unknown" //nolint:gochecknoglobals // Populated by GoReleaser through ldflags.
	Branch    = "unknown" //nolint:gochecknoglobals // Populated by GoReleaser through ldflags.
	BuildDate = "unknown" //nolint:gochecknoglobals // Populated by GoReleaser through ldflags.
)

// String returns the complete build version description.
func String() string {
	return fmt.Sprintf(
		"%s (revision: %s, branch: %s, built: %s)",
		Version,
		Revision,
		Branch,
		BuildDate,
	)
}
