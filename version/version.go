// Package version contains the release version shared by all apipig binaries.
package version

// Default is used for local builds that do not provide a linker override.
const Default = "1.0.0"

// Version is replaced by release builds through:
// -ldflags "-X apipig/version.Version=<release-version>"
var Version = Default
