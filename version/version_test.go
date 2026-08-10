package version

import (
	"flag"
	"os"
	"testing"
)

var expectedVersion = flag.String("expected-version", "", "expected linked release version")

func TestVersion(t *testing.T) {
	want := *expectedVersion
	if want == "" {
		want = os.Getenv("APIPIG_EXPECTED_VERSION")
	}
	if want == "" {
		want = Default
	}
	if Version != want {
		t.Fatalf("Version = %q, want %q", Version, want)
	}
}
