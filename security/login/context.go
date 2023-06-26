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
		flow    string
		entries []ModuleEntry
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

	if err := c.loadModules(handler); err != nil {
		return promise.Reject[security.Subject](Error{err})
	}

	subject := security.NewSubject()

	return promise.New(func(resolve func(security.Subject), reject func(error)) {
		for _, mod := range c.modules {
			err := mod.Login(subject, handler)
			if err != nil {
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

func (c *Context) loadModules(
	handler miruken.Handler,
) error {
	if flow := c.flow; flow != "" {
		f, _, err := provides.Type[Flow](handler, &config.Load{Path: flow})
		if err != nil {
			return err
		} else if len(f) == 0 {
			return fmt.Errorf("no modules found in flow %q", flow)
		}
		c.entries = f
	}
	modules := make([]Module, len(c.entries), len(c.entries))

	for i, entry := range c.entries {
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


// NewFlow creates a Context from the configured flow.
// `flow` is used as the path into the application configuration.
func NewFlow(flow string) *Context {
	return &Context{flow: flow}
}

// New creates a Context from the supplied module entries.
func New(modules ...ModuleEntry) *Context {
	if len(modules) == 0 {
		panic("login: at least one module is required")
	}
	return &Context{entries: modules}
}
