package buildinfo

// Version, Commit, and BuiltAt are populated via -ldflags at build time. They
// default to development-friendly values when not provided.
//
// Example:
//   go build -ldflags "\
//     -X 'caatsm/internal/infra/buildinfo.Version=v0.4.3' \
//     -X 'caatsm/internal/infra/buildinfo.Commit=abc1234' \
//     -X 'caatsm/internal/infra/buildinfo.BuiltAt=2025-11-16T08:35:00Z' \
//   "

var (
	Version = "dev"
	Commit  = "unknown"
	BuiltAt = ""
)


