package auth

import (
	"net/http"
	"strings"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/login"
)

type (
	// Scheme binds a http request to a login flow.
	Scheme interface {
		Accept(*http.Request) (miruken.Handler, error, bool)
		Challenge(http.ResponseWriter, *http.Request, error) int
	}

	// Options configures authentication middleware.
	Options struct {
		flows    []flowSpec
		required bool
	}

	// Authentication applies login flows to auth requests.
	Authentication struct {
		options Options
	}

	// FlowBuilder configures a login flow.
	FlowBuilder struct {
		a    *Authentication
		flow flowSpec
	}

	flowSpec struct {
		ref    string
		flow   login.Flow
		scheme Scheme
	}
)

func (b *FlowBuilder) Scheme(scheme Scheme) *Authentication {
	if internal.IsNil(scheme) {
		panic("scheme cannot be nil")
	}
	b.flow.scheme = scheme
	b.a.options.flows = append(b.a.options.flows, b.flow)
	return b.a
}

func (a *Authentication) Constructor(
	_ *struct{ args.Optional }, options Options,
) {
	a.options = options
}

func (a *Authentication) WithFlowAlias(alias string) *FlowBuilder {
	if alias == "" {
		panic("flow alias cannot be empty")
	}
	return &FlowBuilder{a: a, flow: flowSpec{ref: alias}}
}

func (a *Authentication) WithFlow(flow login.Flow) *FlowBuilder {
	if len(flow) == 0 {
		panic("flow cannot be empty")
	}
	return &FlowBuilder{a: a, flow: flowSpec{flow: flow}}
}

func (a *Authentication) Required() *Authentication {
	a.options.required = true
	return a
}

func (a *Authentication) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	n func(miruken.Handler),
) {
	for _, flow := range a.options.flows {
		scheme := flow.scheme
		if ch, err, ok := scheme.Accept(r); ok {
			if err != nil {
				sc := scheme.Challenge(w, r, err)
				w.WriteHeader(sc)
				return
			}
			var ctx *login.Context
			if flow.flow != nil {
				ctx = login.NewFlow(flow.flow)
			} else {
				ctx = login.New(flow.ref)
			}
			lh := miruken.AddHandlers(h, ch)
			ps := ctx.Login(lh)
			if sub, err := ps.Await(); err == nil {
				if sub.Authenticated() {
					sub.AddCredentials(scheme)
					n(miruken.BuildUp(h, provides.With(sub)))
					ctx.Logout(lh)
					return
				}
			} else {
				statusCode := scheme.Challenge(w, r, err)
				w.WriteHeader(statusCode)
				return
			}
		}
	}

	// Return unauthorized if authentication is required.
	if a.options.required {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Provide an unauthenticated subject.
	n(miruken.BuildUp(h, provides.With(security.NewSubject())))
}

// WriteWWWAuthenticateHeader writes the `WWW-Authenticate`
// http response header for the supplied scheme.
func WriteWWWAuthenticateHeader(
	w      http.ResponseWriter,
	scheme string,
	realm  string,
	params map[string]string,
	err    error,
) {
	if scheme == "" {
		panic("scheme is required")
	}
	var h strings.Builder
	h.WriteString(scheme)
	if realm != "" {
		h.WriteString(" realm=\"")
		h.WriteString(realm)
		h.WriteString("\"")
	}
	for k, v := range params {
		if h.Len() > len(scheme) {
			h.WriteString(",")
		}
		h.WriteString(" ")
		h.WriteString(k)
		h.WriteString("=\"")
		h.WriteString(v)
		h.WriteString("\"")
	}
	if _, ok := params["error_description"]; !ok {
		if err != nil {
			if h.Len() > len(scheme) {
				h.WriteString(",")
			}
			var errDesc string
			switch e := err.(type) {
			case login.Error:
				errDesc = e.Cause.Error()
			default:
				errDesc = e.Error()
			}
			if errDesc != "" {
				h.WriteString(" error_description=\"")
				h.WriteString(errDesc)
				h.WriteString("\"")
			}
		}
	}
	w.Header().Add("WWW-Authenticate", h.String())
}

// WithFlowAlias starts a new authentication flow builder
// with an alias to a login flow.
func WithFlowAlias(flow string) *FlowBuilder {
	auth := &Authentication{}
	return auth.WithFlowAlias(flow)
}

// WithFlow starts a new authentication flow builder
// with the definition of a login flow.
func WithFlow(flow login.Flow) *FlowBuilder {
	auth := &Authentication{}
	return auth.WithFlow(flow)
}
