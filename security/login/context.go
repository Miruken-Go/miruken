package login

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/config"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/security"
)

type (
	// Context coordinates the entire login process.
	Context struct {
		flow    Flow
		path    string
		modules []Module
		subject security.Subject
	}

	// Error reports a failure during the login process.
	Error struct {
		Cause error
	}
)


// Error

func (e Error) Error() string {
	if miruken.IsNil(e.Cause) {
		return "login failed"
	}
	return fmt.Sprintf("login failed: %s", e.Cause.Error())
}

func (e Error) Unwrap() error {
	return e.Cause
}


// Context

// Login performs the authentication.
// returns the authenticated security.Subject or
// and error if authentication failed.
func (c *Context) Login(
	handler miruken.Handler,
) *promise.Promise[security.Subject] {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}

	// Already authenticated?
	if !miruken.IsNil(c.subject) {
		return promise.Resolve(c.subject)
	}

	// Initialize flow modules
	if err := c.initFlow(handler); err != nil {
		return promise.Reject[security.Subject](Error{err})
	}

	return promise.New(func(resolve func(security.Subject), reject func(error)) {
		subject := security.NewSubject()
		for i, mod := range c.modules {
			err := mod.Login(subject, handler)
			if err != nil {
				for ii := i-1; ii >= 0; ii-- {
					// clear successful modules
					_ = c.modules[ii].Logout(subject, handler)
				}
				reject(Error{err})
				return
			}
		}
		c.subject = subject
		resolve(subject)
	})
}

// Logout the security.Subject.
func (c *Context) Logout(
	handler miruken.Handler,
) *promise.Promise[security.Subject] {
	subject := c.subject
	if miruken.IsNil(subject) {
		return promise.Reject[security.Subject](
			Error{errors.New("login must succeed first")})
	}
	return promise.New(func(resolve func(security.Subject), reject func(error)) {
		for _, mod := range c.modules {
			err := mod.Logout(subject, handler)
			if err != nil {
				reject(Error{err})
				return
			}
		}
		resolve(c.subject)
	})
}

func (c *Context) initFlow(
	handler miruken.Handler,
) error {
	if c.modules != nil {
		return nil
	}
	if flow := c.path; flow != "" {
		f, _, err := provides.Type[Flow](handler, &config.Load{Path: flow})
		if err != nil {
			return err
		} else if len(f) == 0 {
			return fmt.Errorf("no modules found in flow %q", flow)
		}
		c.flow = f
	}
	modules := make([]Module, len(c.flow), len(c.flow))

	for i, entry := range c.flow {
		m, mp, err := creates.Key[Module](handler, entry.Module)
		if err != nil {
			return err
		} else if mp != nil {
			if m, err = mp.Await(); err != nil {
				return err
			}
		}
		if mi, ok := m.(interface{
			Init(map[string]any) error
		}); ok {
			opts := entry.Options
			if opts == nil {
				opts = map[string]any{}
			}
			if err := mi.Init(opts); err != nil {
				return err
			}
		}
		modules[i] = m
	}

	c.modules = modules
	return nil
}


// New creates a Context from the configured flow path.
// `flow` is used as the path into the application configuration.
func New(flow string) *Context {
	if flow == "" {
		panic("login: path cannot be empty")
	}
	return &Context{path: flow}
}

// NewFlow creates a Context from the supplied flow entries.
func NewFlow(flow Flow) *Context {
	if len(flow) == 0 {
		panic("login: at least one module is required")
	}
	return &Context{flow: flow}
}
