package httpsrv

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/login/callback"
	"net/http"
	"strings"
)

type (
	authenticate struct {
		flow    string
		handler miruken.Handler
		next    http.Handler
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


func (l *authenticate) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		token := strings.Split(auth, "Bearer ")
		if len(token) != 2 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "401 malformed token", http.StatusUnauthorized)
			return
		}
		ctx := login.NewFlow(l.flow)
		ch  := &callbackHandler{token[1]}
		ps  := ctx.Login(miruken.AddHandlers(l.handler, ch))
		if sub, err := ps.Await(); err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			fmt.Println(sub)
		}
	}
	l.next.ServeHTTP(w, r)
}

func Authenticate(
	next    http.Handler,
	flow    string,
	handler miruken.Handler,
) http.Handler {
	if miruken.IsNil(handler) {
		panic("next cannot be nil")
	}
	if flow == "" {
		panic("flow cannot be empty")
	}
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	return &authenticate{flow, handler, next}
}
