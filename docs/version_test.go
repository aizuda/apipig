package docs

import (
	buildversion "apipig/version"
	"testing"
)

func TestSwaggerInfoUsesBuildVersion(t *testing.T) {
	if SwaggerInfo.Version != buildversion.Version {
		t.Fatalf("Swagger version = %q, want %q", SwaggerInfo.Version, buildversion.Version)
	}
}
