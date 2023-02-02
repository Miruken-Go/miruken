package openapi

import (
	"net/http"
)

// Handler is a factory method that generates an api http.HandlerFunc.
// if enableCors is true, then the handler will generate cors headers.
func (i *Installer) Handler(enableCors bool) http.HandlerFunc {
	/*
	mux := &sync.Mutex{}
	byHostAndScheme := map[string]*API{}

	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if enableCors {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, api_key, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT")
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.WriteHeader(http.StatusOK)

		// customize the swagger header based on host
		//
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
			v = i.clone()
			v.Host = req.Host
			v.Schemes = []string{scheme}
			byHostAndScheme[hostAndScheme] = v
		}
		mux.Unlock()

		json.NewEncoder(w).Encode(v)
	}
	 */
	return nil
}
