package web

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	appDocs "github.com/mococa/gymshark-packs/docs"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html><head>
  <title>Pack Calculator API</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="/docs/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="/docs/swagger-ui-bundle.js"></script>
<script src="/docs/swagger-ui-standalone-preset.js"></script>
<script>
window.onload = function() {
  SwaggerUIBundle({
    url: "/docs/doc.json",
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    layout: "StandaloneLayout"
  })
}
</script>
</body></html>`

func SwaggerIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerUIHTML))
}

func SwaggerSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(appDocs.SwaggerInfo.ReadDoc()))
}

func SwaggerAssets() http.Handler {
	return http.StripPrefix("/docs", httpSwagger.WrapHandler)
}
