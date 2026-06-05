package main

import (
	"net/http"

	"github.com/swaggo/swag"
)

const scalarHTML = `<!doctype html>
<html>
<head>
  <title>Vigil API</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
  <script id="api-reference" data-url="/v1/openapi.json"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

func (app *application) openAPIHandler(w http.ResponseWriter, r *http.Request) {
	spec, err := swag.ReadDoc()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if app.config.env == "development" {
		w.Header().Set("Cache-Control", "no-store")
	}
	_, _ = w.Write([]byte(spec))
}

func (app *application) docsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(scalarHTML))
}
