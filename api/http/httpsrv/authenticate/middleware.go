package authenticate

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security/login"
	"net/http"
)

type (
	// Scheme binds a http request to a login flow.
	Scheme interface {
		Accept(r *http.Request) (miruken.Handler, error, bool)
		Challenge(w http.ResponseWriter, r *http.Request, err error) int
		Forbid(w http.ResponseWriter, r *http.Request, err error)
	}

	// Authentication applies login flows to authenticate requests.
	Authentication struct {
		flows []flowSpec
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
	if miruken.IsNil(scheme) {
		panic("schema cannot be nil")
	}
	b.flow.scheme = scheme
	b.a.flows = append(b.a.flows, b.flow)
	return b.a
}


func (a *Authentication) FlowRef(ref string) *FlowBuilder {
	if ref == "" {
		panic("flow ref cannot be empty")
	}
	return &FlowBuilder{a: a, flow: flowSpec{ref: ref}}
}

func (a *Authentication) Flow(flow login.Flow) *FlowBuilder {
	if len(flow) == 0 {
		panic("flow cannot be empty")
	}
	return &FlowBuilder{a: a, flow: flowSpec{flow: flow}}
}

func (a *Authentication) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	h miruken.Handler,
	m httpsrv.Middleware,
	n func(miruken.Handler),
) {
	// Merge explicit flows.
	flows := a.flows
	if ma, ok := m.(*Authentication); ok {
		if ma.flows != nil {
			flows = ma.flows
		}
	}

	// Skip authentication if no flows configured.
	// handlers requiring a security.Subject will not execute.
	if len(flows) == 0 {
		n(h)
		return
	}

	for _, flow := range flows {
		if ch, err, ok := flow.scheme.Accept(r); ok {
			if err != nil {
				statusCode := flow.scheme.Challenge(w, r, err)
				w.WriteHeader(statusCode)
				return
			}
			var ctx *login.Context
			if flow.flow != nil {
				ctx = login.NewFlow(flow.flow)
			} else {
				ctx = login.New(flow.ref)
			}
			ps := ctx.Login(miruken.AddHandlers(h, ch))
			if sub, err := ps.Await(); err == nil {
				n(miruken.BuildUp(h, provides.With(sub)))
			} else {
				statusCode := flow.scheme.Challenge(w, r, err)
				w.WriteHeader(statusCode)
			}
			return
		}
	}

	// Generate challenge for all flows.
	for _, flow := range flows {
		flow.scheme.Challenge(w, r, nil)
	}
	w.WriteHeader(http.StatusUnauthorized)
}


// WithFlowRef returns new authentication middleware with
// the initial login flow reference.
func WithFlowRef(flow string) *FlowBuilder {
	auth := &Authentication{}
	return auth.FlowRef(flow)
}

// WithFlow returns new authentication middleware with
// the initial login flow definition.
func WithFlow(flow login.Flow) *FlowBuilder {
	auth := &Authentication{}
	return auth.Flow(flow)
}
