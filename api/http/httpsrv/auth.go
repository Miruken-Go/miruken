package httpsrv

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"net/http"
	"strings"
)

type (
	// Authenticate provides Middleware to authenticate
	// a http request for the given login flow.
	Authenticate struct {
		Flow string
	}

	callbackHandler struct {
		token string
	}
)


func (h *callbackHandler) Handle(
	c        any,
	greedy   bool,
	composer miruken.Handler,
) miruken.HandleResult {
	if n, ok := c.(*callback.Name); ok {
		n.SetName(h.token)
		return miruken.Handled
	}
	return miruken.NotHandled
}


func (a *Authenticate) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	m Middleware,
	n func(handler miruken.Handler),
) {
	auth := r.Header.Get("Authorization")
	// if no 'Authorization' header is present, skip authentication.
	// handlers requiring a security.Subject will not execute.
	if auth != "" {
		token := strings.Split(auth, "Bearer ")
		if len(token) != 2 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "401 malformed token", http.StatusUnauthorized)
			return
		}
		flow := a.Flow
		if ma, ok := m.(*Authenticate); ok {
			if ma.Flow != "" {
				flow = ma.Flow
			}
		}
		ctx := login.NewFlow(flow)
		ch  := &callbackHandler{token[1]}
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

