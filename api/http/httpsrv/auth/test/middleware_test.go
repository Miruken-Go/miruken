package test

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/api/http/httpsrv/auth"
	"github.com/miruken-go/miruken/internal/slices"
	"github.com/miruken-go/miruken/security"
	"github.com/miruken-go/miruken/security/login"
	"github.com/miruken-go/miruken/security/password"
	"github.com/miruken-go/miruken/security/principal"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MiddlewareTestSuite struct {
	suite.Suite
}

func (suite *MiddlewareTestSuite) Setup(specs ...any) miruken.Handler {
	handler, _ := miruken.Setup(password.Feature()).Specs(specs...).Handler()
	return handler
}

func (suite *MiddlewareTestSuite) TestHandler() {
	suite.Run("Authorize", func() {
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request, sub security.Subject) {
				user := slices.OfType[security.Principal, principal.User](sub.Principals())
				_, _ = fmt.Fprintf(w, "Hello %s", user[0])
			}, auth.WithFlow([]login.ModuleEntry{
				{ Module: "login.pwd", Options: map[string]any{
					"credentials": map[string]any{
						"user": "password",
					},
				},
			}}).Basic())
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		req.SetBasicAuth("user", "password")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello user", string(body))
	})

	suite.Run("Anonymous", func() {
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request, sub security.Subject) {
				user := slices.OfType[security.Principal, principal.User](sub.Principals())
				suite.Len(user, 0)
				_, _ = fmt.Fprint(w, "Hello World")
			}, auth.WithFlow([]login.ModuleEntry{
				{ Module: "login.pwd", Options: map[string]any{
					"credentials": map[string]any{
						"user": "foo",
					},
				},
				}}).Basic())
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		req.SetBasicAuth("user", "password")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello World", string(body))
	})

	suite.Run("Deny", func() {
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request, sub security.Subject) {
				_, _ = fmt.Fprint(w, "Hello World")
			}, auth.WithFlow([]login.ModuleEntry{
				{ Module: "login.pwd", Options: map[string]any{
					"credentials": map[string]any{
						"user": "password",
					},
				},
				}}).Basic().Required())
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		req.SetBasicAuth("user", "foo")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(401, resp.StatusCode)
	})
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}