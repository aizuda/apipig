package docs

import buildversion "apipig/version"

// Keep the generated Swagger source stable while exposing the linked release
// version at runtime. This survives subsequent swag init runs.
func init() {
	SwaggerInfo.Version = buildversion.Version
}
