package openapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

func Handler(docs map[string]*openapi3.T, enableCors bool) http.HandlerFunc {
	mux := &sync.Mutex{}
	byHostAndScheme := map[string]openapi3.T{}

	return func(w http.ResponseWriter, req *http.Request) {
		var doc *openapi3.T
		mod := req.URL.Query().Get("mod")
		if mod == "" {
			if len(docs) > 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			for _, doc = range docs {
				break
			}
		} else {
			mod, _ = url.QueryUnescape(mod)
			doc = docs[mod]
		}

		if doc == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if enableCors {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, api_key, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT")
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.WriteHeader(http.StatusOK)

		// customize the document based on host
		scheme := ""
		if req.TLS != nil {
			scheme = "https"
		}
		if v := req.Header.Get("X-Forwarded-Proto"); v != "" {
			scheme = v
		}
		if scheme == "" {
			scheme = req.URL.Scheme
		}
		if scheme == "" {
			scheme = "http"
		}

		hostAndScheme := mod + "#" + req.Host + ":" + scheme
		mux.Lock()
		v, ok := byHostAndScheme[hostAndScheme]
		if !ok {
			v = *doc
			uri := url.URL{
				Scheme: scheme,
				Host:   req.Host,
			}
			v.Servers = openapi3.Servers{
				&openapi3.Server{
					URL: uri.String(),
				},
			}
			byHostAndScheme[hostAndScheme] = v
		}
		mux.Unlock()

		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		_ = enc.Encode(v)
	}
}
