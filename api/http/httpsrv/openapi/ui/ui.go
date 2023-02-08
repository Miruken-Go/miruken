package ui

import (
	"embed"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/miruken-go/miruken/api/http/httpsrv/openapi"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var static embed.FS

func Handler(prefix string, api *openapi3.T) http.HandlerFunc {
	dir, _ := fs.Sub(static, "static")
	server := http.StripPrefix(prefix, http.FileServer(http.FS(dir)))
	return func(rw http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "swagger.json") {
			openapi.Handler(api, false)(rw, r)
			return
		}
		server.ServeHTTP(rw, r)
	}
}