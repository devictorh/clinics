// Package docs embarca os artefatos de documentação servidos pela API.
package docs

import _ "embed"

// OpenAPISpec é o contrato OpenAPI 3.0 da API, fonte da verdade servida
// em /openapi.yaml e renderizada pelo Swagger UI em /docs.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
