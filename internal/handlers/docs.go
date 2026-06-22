package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/lupppig/forge-vod/internal/api"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Forge VOD API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: "/openapi.json",
        dom_id: "#swagger-ui",
      });
    };
  </script>
</body>
</html>`

// DocsHandler serves the Swagger UI page.
func DocsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerUIHTML))
}

// OpenAPIHandler serves the embedded OpenAPI spec as JSON.
func OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	spec, err := api.GetSwagger()
	if err != nil {
		http.Error(w, "failed to load spec", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(spec)
}
