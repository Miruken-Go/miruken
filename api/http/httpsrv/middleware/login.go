package middleware

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"net/http"
	"strings"
)

// Login provides Middleware to authenticate
// a http request for a given login flow.
type Login struct {
	Flow string
}


func (l *Login) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	m httpsrv.Middleware,
	n func(handler miruken.Handler),
) {
	auth := r.Header.Get("Authorization")
	// if no 'Authorization' header is present, skip authentication.
	// handlers requiring a security.Subject will not execute.
	if auth != "" {
		token := strings.Split(auth, "Bearer ")
		if len(token) != 2 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		flow := l.Flow
		if ma, ok := m.(*Login); ok {
			if ma.Flow != "" {
				flow = ma.Flow
			}
		}
		ctx := login.NewFlow(flow)
		ch  := callback.NameHandler{Name: token[1]}
		ps  := ctx.Login(miruken.AddHandlers(h, ch))
		if sub, err := ps.Await(); err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			h = miruken.BuildUp(h, provides.With(sub))
		}
	}
	n(h)
}

