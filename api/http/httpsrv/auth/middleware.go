package auth

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/internal"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/login"
	"net/http"
	"strings"
)

type (
	// Scheme binds a http request to a login flow.
	Scheme interface {
		Accept(*http.Request) (miruken.Handler, error, bool)
		Challenge(http.ResponseWriter, *http.Request, error) int
	}

	// Authentication applies login flows to auth requests.
	Authentication struct {
		flows    []flowSpec
		required bool
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
	b.a.flows = append(b.a.flows, b.flow)
	return b.a
}


func (a *Authentication) WithFlowRef(ref string) *FlowBuilder {
	if ref == "" {
		panic("flow ref cannot be empty")
	}
	return &FlowBuilder{a: a, flow: flowSpec{ref: ref}}
}

func (a *Authentication) WithFlow(flow login.Flow) *FlowBuilder {
	if len(flow) == 0 {
		panic("flow cannot be empty")
	}
	return &FlowBuilder{a: a, flow: flowSpec{flow: flow}}
}

func (a *Authentication) Required() *Authentication {
	a.required = true
	return a
}


func (a *Authentication) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	m httpsrv.Middleware,
	h miruken.Handler,
	n httpsrv.Handler,
) error {
	// Merge explicit flows.
	flows := a.flows
	if ma, ok := m.(*Authentication); ok {
		if ma.flows != nil {
			flows = ma.flows
		}
	}

	for _, flow := range flows {
		scheme := flow.scheme
		if ch, err, ok := scheme.Accept(r); ok {
			if err != nil {
				sc := scheme.Challenge(w, r, err)
				w.WriteHeader(sc)
				return nil
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
				sub.AddCredentials(scheme)
				n.ServeHTTP(w, r, miruken.BuildUp(h, provides.With(sub)))
				ctx.Logout(lh)
			} else {
				statusCode := scheme.Challenge(w, r, err)
				w.WriteHeader(statusCode)
			}
			return nil
		}
	}

	// Return unauthorized if authentication is required.
	if a.required {
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}

	// Provide an unauthenticated subject.
	n.ServeHTTP(w, r, miruken.BuildUp(h, provides.With(security.NewSubject())))
	return nil
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
	for k,v := range params {
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


// WithFlowRef starts a new authentication flow builder
// with a reference to a login flow.
func WithFlowRef(flow string) *FlowBuilder {
	auth := &Authentication{}
	return auth.WithFlowRef(flow)
}

// WithFlow starts a new authentication flow builder
// with the definition of a login flow.
func WithFlow(flow login.Flow) *FlowBuilder {
	auth := &Authentication{}
	return auth.WithFlow(flow)
}
