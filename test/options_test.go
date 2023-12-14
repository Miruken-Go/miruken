package test

import (
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/args"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type (
	FooOptionsHandler struct{}
	FooOptions        struct {
		Increment int
	}
)

func (h *FooOptionsHandler) HandleFoo(
	_ *handles.It, foo *Foo,
	_ *struct{ args.FromOptions }, options FooOptions,
) {
	for i := 0; i < options.Increment; i++ {
		foo.Inc()
	}
}

type OptionsTestSuite struct {
	suite.Suite
}

func (suite *OptionsTestSuite) TestOptions() {
	type Header struct {
		key   string
		value any
	}

	type ServerOptions struct {
		Url       string
		Timeout   int
		KeepAlive miruken.Option[bool]
		Headers   []Header
	}

	suite.Run("Inline", func() {
		handler, _ := setup.New().
			Options(ServerOptions{
				Url:     "https://playsoccer.com",
				Timeout: 30,
				Headers: []Header{
					{"Content-Key", "application/json"},
					{"Content-Encoding", "compress"},
				}}).
			Context()
		options, ok := miruken.GetOptions[ServerOptions](handler)
		suite.True(ok)
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
		suite.ElementsMatch([]Header{
			{"Content-Key", "application/json"},
			{"Content-Encoding", "compress"}}, options.Headers)
	})

	suite.Run("InlineWithOptionsPtr", func() {
		serverOpt := new(ServerOptions)
		serverOpt.Url = "https://playsoccer.com"
		serverOpt.Timeout = 30
		handler, _ := setup.New().
			Options(serverOpt).
			Context()
		options, ok := miruken.GetOptions[ServerOptions](handler)
		suite.True(ok)
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("Creates", func() {
		handler, _ := setup.New().
			Options(ServerOptions{
				Url:     "https://playsoccer.com",
				Timeout: 30,
			}).Context()
		var options *ServerOptions
		suite.True(miruken.GetOptionsInto(handler, &options))
		suite.NotNil(options)
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("MergesInline", func() {
		handler, _ := setup.New().
			Options(ServerOptions{
				Url:     "https://playsoccer.com",
				Timeout: 30,
			}).Context()
		options := ServerOptions{Timeout: 60}
		suite.True(miruken.GetOptionsInto(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(60, options.Timeout)
	})

	suite.Run("MergesCreate", func() {
		handler, _ := setup.New().
			Options(ServerOptions{
				Url:     "https://playsoccer.com",
				Timeout: 30,
			}).Context()
		options := new(ServerOptions)
		options.Url = "https://improving.com"
		suite.True(miruken.GetOptionsInto(handler, options))
		suite.NotNil(options)
		suite.Equal("https://improving.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("Combines", func() {
		handler, _ := setup.New().
			Options(ServerOptions{
				Url:       "https://directv.com",
				KeepAlive: miruken.Set(true),
			}, ServerOptions{
				Timeout:   60,
				KeepAlive: miruken.Set(false),
			}).Context()
		options, ok := miruken.GetOptions[ServerOptions](handler)
		suite.True(ok)
		suite.Equal("https://directv.com", options.Url)
		suite.Equal(60, options.Timeout)
		suite.Equal(miruken.Set(false), options.KeepAlive)
	})

	suite.Run("AppendsSlice", func() {
		handler, _ := setup.New().
			Options(ServerOptions{
				Url: "https://netflix.com",
				Headers: []Header{
					{"Content-Key", "application/json"},
					{"Authorization", "Bearer j23j2eh323"},
				},
			}, ServerOptions{
				Timeout: 100,
				Headers: []Header{
					{"Content-Encoding", "compress"},
				},
			}).Context()
		options, ok := miruken.GetOptions[ServerOptions](handler)
		suite.True(ok)
		suite.Equal("https://netflix.com", options.Url)
		suite.Equal(100, options.Timeout)
		suite.ElementsMatch([]Header{
			{"Content-Key", "application/json"},
			{"Authorization", "Bearer j23j2eh323"},
			{"Content-Encoding", "compress"}}, options.Headers)
	})

	suite.Run("NoMatch", func() {
		handler, _ := setup.New().Context()
		options, ok := miruken.GetOptions[ServerOptions](handler)
		suite.False(ok)
		suite.Equal("", options.Url)
		suite.Equal(0, options.Timeout)
	})

	suite.Run("NoMatchCreate", func() {
		handler, _ := setup.New().Context()
		var options *ServerOptions
		suite.False(miruken.GetOptionsInto(handler, &options))
		suite.Nil(options)
	})

	suite.Run("FromOptions", func() {
		handler, _ := setup.New().
			Specs(&FooOptionsHandler{}).
			Options(FooOptions{2}).
			Context()
		foo := new(Foo)
		result := handler.Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(2, foo.Count())
	})

	suite.Run("Errors", func() {
		suite.Run("Nil Options", func() {
			defer func() {
				suite.Equal("options cannot be nil", recover())
			}()
			miruken.Options(nil)
		})

		suite.Run("Not Struct Options", func() {
			defer func() {
				suite.Equal("options must be a struct or *struct", recover())
			}()
			miruken.Options(1)
		})
	})
}

func TestOptionsTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsTestSuite))
}
