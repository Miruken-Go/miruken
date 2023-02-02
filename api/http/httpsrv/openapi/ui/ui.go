package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var static embed.FS

func Handler(prefix string, /*api *swagger.API*/) http.HandlerFunc {
	dir, _ := fs.Sub(static, "static")
	server := http.StripPrefix(prefix, http.FileServer(http.FS(dir)))
	return func(rw http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "swagger.json") {
			//api.Handler(false)(rw, r)
			return
		}
		server.ServeHTTP(rw, r)
	}
}