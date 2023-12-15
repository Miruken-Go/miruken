package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-logr/logr"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api/http/httpsrv"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/logs"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type (
	SayHello struct{}

	WeatherUnit    uint
	WeatherOptions struct {
		Unit WeatherUnit
	}
	GetWeather struct {
		opts WeatherOptions
	}

	Fibonacci struct{}
)

const (
	WeatherUnitFahrenheit = iota
	WeatherUnitCelsius
)

func (h SayHello) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	_, _ = fmt.Fprintf(w, "Hello %s", r.URL.Query().Get("name"))
}

func (u WeatherUnit) FromFahrenheit(f float64) float64 {
	switch u {
	case WeatherUnitFahrenheit:
		return f
	case WeatherUnitCelsius:
		return (f - 32) * 5 / 9
	default:
		panic(fmt.Sprintf("Unknown unit %d", u))
	}
}

func (u WeatherUnit) Abbrev() string {
	switch u {
	case WeatherUnitFahrenheit:
		return "F"
	case WeatherUnitCelsius:
		return "C"
	default:
		panic(fmt.Sprintf("Unknown unit %d", u))
	}
}

func (h *GetWeather) Constructor(
	_ *struct{ args.Optional }, options WeatherOptions,
) {
	h.opts = options
}

func (h *GetWeather) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
	c miruken.Handler,
) {
	opt := h.opts
	zip := r.URL.Query().Get("zip")
	switch zip {
	case "75032":
		_, _ = fmt.Fprintf(w, "The weather in Health TX is %.1f°%s",
			opt.Unit.FromFahrenheit(76), opt.Unit.Abbrev())
	case "11580":
		_, _ = fmt.Fprintf(w, "The weather in Valley Stream NY is %.1f°%s",
			opt.Unit.FromFahrenheit(44), opt.Unit.Abbrev())
	case "90011":
		_, _ = fmt.Fprintf(w, "The weather in Los Angeles CA is %.1f°%s",
			opt.Unit.FromFahrenheit(73), opt.Unit.Abbrev())
	default:
		_, _ = fmt.Fprintf(w, "Unknown weather for %s", zip)
	}
}

func (h Fibonacci) Calculate(
	w http.ResponseWriter,
	r *http.Request,
	log logr.Logger,
) {
	n, err := strconv.Atoi(r.URL.Query().Get("n"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		fib := fibonacci(n)
		log.Info("Fibonacci", "n", n, "value", fib)
		_, _ = fmt.Fprintf(w, "Fibonacci(%d) = %d", n, fib)
	}
}

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

type HandlerTestSuite struct {
	suite.Suite
}

func (suite *HandlerTestSuite) Setup(specs ...any) *context.Context {
	ctx, _ := setup.New(logs.Feature(NewStdoutLogger())).
		Specs(specs...).
		Context()
	return ctx
}

func (suite *HandlerTestSuite) TestHandler() {
	suite.Run("Function", func() {
		handler := httpsrv.Use(suite.Setup(),
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, "Hello World")
			})
		req := httptest.NewRequest("GET", "http://hello.com", http.NoBody)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello World", string(body))
	})

	suite.Run("Context", func() {
		handler := httpsrv.Use(suite.Setup(),
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, "Hello Goodbye")
			}))
		req := httptest.NewRequest("GET", "http://hello.com", http.NoBody)
		w := httptest.NewRecorder()
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
		req := httptest.NewRequest("GET", "http://hello.com", http.NoBody)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello World", string(body))
	})

	suite.Run("Extended Context", func() {
		handler := httpsrv.Use(suite.Setup(),
			httpsrv.HandlerFunc(func(w http.ResponseWriter, r *http.Request, c miruken.Handler) {
				suite.NotNil(c)
				_, _ = fmt.Fprint(w, "Hello Goodbye")
			}))
		req := httptest.NewRequest("GET", "http://hello.com", nil)
		w := httptest.NewRecorder()
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
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		suite.Equal(200, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		suite.Equal("Hello World", string(body))
	})

	suite.Run("Resolve", func() {
		suite.Run("Context", func() {
			handler := httpsrv.Use(suite.Setup(SayHello{}), httpsrv.H[SayHello]())
			req := httptest.NewRequest("GET", "http://hello.com?name=Craig", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("Hello Craig", string(body))
		})

		suite.Run("Extended Context", func() {
			handler := httpsrv.Use(suite.Setup(&GetWeather{}), httpsrv.H[*GetWeather]())
			req := httptest.NewRequest("GET", "http://weather.com?zip=11580", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("The weather in Valley Stream NY is 44.0°F", string(body))
		})

		suite.Run("Extended Context with Options", func() {
			handler := httpsrv.Use(suite.Setup(&GetWeather{}),
				httpsrv.H[*GetWeather](WeatherOptions{Unit: WeatherUnitCelsius}))
			req := httptest.NewRequest("GET", "http://weather.com?zip=11580", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("The weather in Valley Stream NY is 6.7°C", string(body))
		})

		suite.Run("Dynamic Context", func() {
			handler := httpsrv.Use(suite.Setup(Fibonacci{}), httpsrv.H[Fibonacci]())
			req := httptest.NewRequest("GET", "http://fibonacci.com?n=5", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			resp := w.Result()
			suite.Equal(200, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			suite.Equal("Fibonacci(5) = 5", string(body))
		})
	})
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
