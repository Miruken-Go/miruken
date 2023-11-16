package test

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/logs"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type (
	SayHello struct {}

	Logger struct {
		httpsrv.MiddlewareAdapter
	}
)


func (h SayHello) ServeHTTP(
	w        http.ResponseWriter,
	r        *http.Request,
	composer miruken.Handler,
) {
	_, _ = fmt.Fprintf(w, "Hello %s", r.URL.Query().Get("name"))
}

func AddHeader(key, value string) httpsrv.Middleware {
	return httpsrv.MiddlewareFunc(func(
		s httpsrv.Middleware,
		w http.ResponseWriter,
		r *http.Request,
		m httpsrv.Middleware,
		h miruken.Handler,
		n func(miruken.Handler),
	) {
		w.Header().Add(key, value)
		n(h)
	})
}

func (l Logger) Log(
	w   http.ResponseWriter,
	r   *http.Request,
	h   miruken.Handler,
	n   func(miruken.Handler),
	log logr.Logger,
) {
	log.Info("ServeHTTP", "method", r.Method, "url", r.URL)
	n(h)
}

func NewStdoutLogger() logr.Logger {
	return funcr.New(func(prefix, args string) {
		if prefix != "" {
			fmt.Printf("%s: %s\n", prefix, args)
		} else {
			fmt.Println(args)
		}
	}, funcr.Options{})
}


type PipelineTestSuite struct {
	suite.Suite
}

func (suite *PipelineTestSuite) Setup(specs ...any) miruken.Handler {
	handler, _ := miruken.Setup(logs.Feature(NewStdoutLogger())).
		Specs(specs...).
		Handler()
	return handler
}

func (suite *PipelineTestSuite) TestHandler() {
	suite.Run("Function", func() {
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, "Hello World")
			})
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		w   := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello World", string(body))
	})

	suite.Run("Handler", func() {
		handler := httpsrv.Use(suite.Setup(),
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, "Hello Goodbye")
			}))
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		w   := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello Goodbye", string(body))
	})

	suite.Run("Extended Function", func() {
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request, c miruken.Handler) {
				suite.NotNil(c)
				_, _ = fmt.Fprint(w, "Hello World")
			})
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		w   := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello World", string(body))
	})

	suite.Run("Extended Handler", func() {
		handler := httpsrv.Use(suite.Setup(),
			httpsrv.HandlerFunc(func(w http.ResponseWriter, r *http.Request, c miruken.Handler) {
				suite.NotNil(c)
				_, _ = fmt.Fprint(w, "Hello Goodbye")
			}))
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		w   := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello Goodbye", string(body))
	})

	suite.Run("Function Dependencies", func() {
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request, logs logr.Logger, h miruken.Handler) {
				suite.NotNil(h)
				logs.Info("Hello World")
				_, _ = fmt.Fprint(w, "Hello World")
			})
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		w   := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello World", string(body))
	})

	suite.Run("Resolve", func() {
		handler := httpsrv.Resolve[SayHello](suite.Setup(SayHello{}))
		req := httptest.NewRequest("GET", "http://hello.com?name=Craig", nil)
		w   := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello Craig", string(body))
	})
}

func (suite *PipelineTestSuite) TestPipeline() {
	suite.Run("Empty", func() {
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request, composer miruken.Handler) {
				_, _ = fmt.Fprint(w, "Hello")
			},
		)
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		w   := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello", string(body))
	})

	suite.Run("Middleware", func() {
		suite.Run("Function", func() {
			handler := httpsrv.Use(suite.Setup(),
				func(w http.ResponseWriter, r *http.Request, composer miruken.Handler) {
					_, _ = fmt.Fprint(w, "Hello")
				}, AddHeader("X-Test", "World"),
			)
			req := httptest.NewRequest("GET", "http://hello.com", nil)
			w   := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("Hello", string(body))
			suite.Equal("World", resp.Header.Get("X-Test"))
		})

		suite.Run("Dynamic", func() {
			handler := httpsrv.Use(suite.Setup(),
				func(w http.ResponseWriter, r *http.Request, composer miruken.Handler) {
					_, _ = fmt.Fprint(w, "Hello")
				}, Logger{},
			)
			req := httptest.NewRequest("GET", "http://hello.com", nil)
			w   := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("Hello", string(body))
		})

		suite.Run("Pipe", func() {
			handler := httpsrv.Use(suite.Setup(),
				func(w http.ResponseWriter, r *http.Request, composer miruken.Handler) {
					_, _ = fmt.Fprint(w, "Hello")
				}, Logger{}, AddHeader("X-Test", "Goodbye"),
			)
			req := httptest.NewRequest("GET", "http://hello.com", nil)
			w   := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("Hello", string(body))
			suite.Equal("Goodbye", resp.Header.Get("X-Test"))
		})
	})
}

func TestPipelineTestSuite(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}