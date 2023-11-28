package test

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/logs"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type (
	Logger struct {
		log logr.Logger
	}
	LoggerDyn struct {}
)

func AddHeader(key, value string) httpsrv.Middleware {
	return httpsrv.MiddlewareFunc(func(
		w http.ResponseWriter,
		r *http.Request,
		h miruken.Handler,
		n func(miruken.Handler),
	) {
		w.Header().Add(key, value)
		n(h)
	})
}


func (l *Logger) Constructor(
	log logr.Logger,
) {
	l.log = log
}

func (l *Logger) ServeHTTP(
	w   http.ResponseWriter,
	r   *http.Request,
	h   miruken.Handler,
	n   func(miruken.Handler),
) {
	l.log.Info("Logger", "method", r.Method, "url", r.URL)
	n(h)
}

func (l LoggerDyn) Log(
	w   http.ResponseWriter,
	r   *http.Request,
	h   miruken.Handler,
	n   func(miruken.Handler),
	log logr.Logger,
) {
	log.Info("LoggerDyn", "method", r.Method, "url", r.URL)
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

func (suite *PipelineTestSuite) Setup(specs ...any) *context.Context {
	handler, _ := miruken.Setup(logs.Feature(NewStdoutLogger())).
		Specs(specs...).
		Handler()
	return context.New(handler)
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
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request, composer miruken.Handler) {
				_, _ = fmt.Fprint(w, "Hello")
			}, &Logger{logr.Discard()},
		)
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		w   := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello", string(body))
	})

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

	suite.Run("Resolve", func() {
		suite.Run("Middleware", func() {
			handler := httpsrv.Use(suite.Setup(&Logger{}),
				func(w http.ResponseWriter, r *http.Request, composer miruken.Handler) {
					_, _ = fmt.Fprint(w, "Hello")
				}, httpsrv.M[*Logger](),
			)
			req := httptest.NewRequest("GET", "http://hello.com", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("Hello", string(body))
		})

		suite.Run("Dynamic", func() {
			handler := httpsrv.Use(suite.Setup(LoggerDyn{}),
				func(w http.ResponseWriter, r *http.Request, composer miruken.Handler) {
					_, _ = fmt.Fprint(w, "Hello")
				}, httpsrv.M[LoggerDyn](),
			)
			req := httptest.NewRequest("GET", "http://hello.com", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("Hello", string(body))
		})
	})

	suite.Run("Pipe", func() {
		handler := httpsrv.Use(suite.Setup(LoggerDyn{}),
			func(w http.ResponseWriter, r *http.Request, composer miruken.Handler) {
				_, _ = fmt.Fprint(w, "Hello")
			}, httpsrv.M[LoggerDyn](),
			   AddHeader("X-Test", "Goodbye"),
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
}

func TestPipelineTestSuite(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}