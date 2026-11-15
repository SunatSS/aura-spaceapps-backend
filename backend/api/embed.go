// Package api embeds the OpenAPI contract and the Swagger UI that renders it,
// so the API documentation works offline from the same binary.
package api

import "embed"

// OpenAPI is api/openapi.yaml — the backend/frontend contract.
//
//go:embed openapi.yaml
var OpenAPI []byte

// Docs holds Swagger UI (swagger-ui-dist 5.35.0, Apache-2.0) and its page.
//
//go:embed docs
var Docs embed.FS
