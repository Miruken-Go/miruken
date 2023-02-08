package openapi

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"net/http"
	"net/url"
	"sync"
)

func Handler(api *openapi3.T, enableCors bool) http.HandlerFunc {
	mux := &sync.Mutex{}
	byHostAndScheme := map[string]openapi3.T{}

	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if enableCors {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, api_key, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT")
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.WriteHeader(http.StatusOK)

		// customize the swagger header based on host
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

		hostAndScheme := req.Host + ":" + scheme
		mux.Lock()
		v, ok := byHostAndScheme[hostAndScheme]
		if !ok {
			v = *api
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

		enc :=json.NewEncoder(w)
		enc.SetIndent("", "    ")
		_ = enc.Encode(v)
	}
}
