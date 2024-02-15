package httpsrv

import (
	"net/http"
	"time"

	"github.com/miruken-go/miruken/internal"
)

// Config provides http.Server configuration.
type Config struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

// New creates a new http.Server with the provided configuration.
func New(
	handler http.Handler,
	config  *Config,
) *http.Server {
	if handler == nil {
		panic("handler cannot be nil")
	}
	if config == nil {
		config = &Config{}
	}
	return &http.Server{
		Addr:              internal.DefaultValue(config.Addr, ":8080"),
		Handler:           handler,
		ReadTimeout:       internal.DefaultValue(config.ReadTimeout, 1*time.Second),
		ReadHeaderTimeout: internal.DefaultValue(config.ReadHeaderTimeout, 1*time.Second),
		WriteTimeout:      internal.DefaultValue(config.WriteTimeout, 2*time.Second),
		IdleTimeout:       internal.DefaultValue(config.IdleTimeout, 30*time.Second),
		MaxHeaderBytes:    internal.DefaultValue(config.MaxHeaderBytes, 1024),
	}
}

// ListenAndServe creates and starts a http.Server with the provided configuration.
func ListenAndServe(
	handler http.Handler,
	config  *Config,
) error {
	return New(handler, nil).ListenAndServe()
}