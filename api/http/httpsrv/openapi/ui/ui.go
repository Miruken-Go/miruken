package ui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/miruken-go/miruken/api/http/httpsrv/openapi"
)

//go:embed static
var static embed.FS

const mirukenMod = "github.com/miruken-go/miruken"

func Handler(prefix string, docs map[string]*openapi3.T, config *openapi.Config) http.HandlerFunc {
	dir, _ := fs.Sub(static, "static")
	server := http.StripPrefix(prefix, http.FileServer(http.FS(dir)))
	h := openapi.Handler(docs, false)
	return func(rw http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "openapi.json") {
			h(rw, r)
		} else if strings.HasSuffix(r.URL.Path, "swagger-initializer.js") {
			rw.Header().Set("Content-Type", "application/javascript")
			_, _ = rw.Write([]byte(`window.onload = function() {
  //<editor-fold desc="Changeable Configuration Block">

  // the following lines will be replaced by docker/configurator, when it runs in a docker-container
  window.ui = SwaggerUIBundle({
    urls: [
`))
			cnt := 0
			var primary string
			for mod := range docs {
				cnt++
				if primary == "" && mod != mirukenMod {
					primary = mod
				}
				_, _ = rw.Write([]byte(`      { url: "`))
				_, _ = rw.Write([]byte("./openapi.json?mod="))
				_, _ = rw.Write([]byte(url.QueryEscape(mod)))
				_, _ = rw.Write([]byte(`", name: "`))
				_, _ = rw.Write([]byte(mod))
				_, _ = rw.Write([]byte(`" }`))
				if cnt < len(docs) {
					_, _ = fmt.Fprintln(rw, ",")
				} else {
					_, _ = fmt.Fprintln(rw)
				}
			}
			_, _ = rw.Write([]byte(`    ],
    "urls.primaryName": "`))
			_, _ = rw.Write([]byte(primary))
			_, _ = fmt.Fprintln(rw, `",`)
			_, _ = rw.Write([]byte(`    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout"
  });
`))
			if clientId := config.ClientId; clientId != "" {
				_, _ = rw.Write([]byte(`
  window.ui.initOAuth({
    clientId: "`))
				_, _ = rw.Write([]byte(clientId))
				_, _ = rw.Write([]byte(`"
})`))
			}

			_, _ = rw.Write([]byte(`	
  //</editor-fold>
};`))
		} else {
			server.ServeHTTP(rw, r)
		}
	}
}
