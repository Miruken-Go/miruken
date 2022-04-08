package test

import (
	"github.com/stretchr/testify/suite"
	"miruken.com/miruken"
	"testing"
)

type (
	FooOptionsHandler struct{}
	FooOptions struct{
		Increment int
	}
)

func (h *FooOptionsHandler) HandleFoo(
	_ *miruken.Handles, foo *Foo,
	_ *struct{ miruken.FromOptions }, options FooOptions,
) {
	for i := 0; i < options.Increment; i++ {
		foo.Inc()
	}
}

type OptionsTestSuite struct {
	suite.Suite
}

func (suite *OptionsTestSuite) TestOptions() {
	type Header struct{
		key   string
		value any
	}

	type ServerOptions struct {
		Url       string
		Timeout   int
		KeepAlive miruken.OptionBool
		Headers   []Header
	}

	suite.Run("Inline", func () {
		handler := miruken.NewRootHandler(miruken.WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
			Headers: []Header{
				{"Content-Key", "application/json"},
				{"Content-Encoding", "compress"},
			},
		}))
		var options ServerOptions
		suite.True(miruken.GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
		suite.ElementsMatch([]Header{
			{"Content-Key", "application/json"},
			{"Content-Encoding", "compress"}}, options.Headers)
	})

	suite.Run("InlineWithOptionsPtr", func () {
		serverOpt := new(ServerOptions)
		serverOpt.Url     = "https://playsoccer.com"
		serverOpt.Timeout = 30
		handler := miruken.NewRootHandler(miruken.WithOptions(serverOpt))
		var options ServerOptions
		suite.True(miruken.GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("Creates", func () {
		handler := miruken.NewRootHandler(miruken.WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
		}))
		var options *ServerOptions
		suite.True( miruken.GetOptions(handler, &options))
		suite.NotNil(options)
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("MergesInline", func () {
		handler := miruken.NewRootHandler(miruken.WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
		}))
		options := ServerOptions{Timeout: 60}
		suite.True(miruken.GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(60, options.Timeout)
	})

	suite.Run("MergesCreate", func () {
		handler := miruken.NewRootHandler(miruken.WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
		}))
		options := new (ServerOptions)
		options.Url = "https://improving.com"
		suite.True(miruken.GetOptions(handler, options))
		suite.NotNil(options)
		suite.Equal("https://improving.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("Combines", func () {
		handler := miruken.NewRootHandler(
			miruken.WithOptions(ServerOptions{
				Url:       "https://directv.com",
				KeepAlive: miruken.OptionTrue,
			}),
			miruken.WithOptions(ServerOptions{
				Timeout:   60,
				KeepAlive: miruken.OptionFalse,
			}))
		var options ServerOptions
		suite.True(miruken.GetOptions(handler, &options))
		suite.Equal("https://directv.com", options.Url)
		suite.Equal(60, options.Timeout)
		suite.False(options.KeepAlive.Bool())
	})

	suite.Run("AppendsSlice", func () {
		handler := miruken.NewRootHandler(
			miruken.WithOptions(ServerOptions{
				Url:"https://netflix.com",
				Headers: []Header{
					{"Content-Key", "application/json"},
					{"Authorization", "Bearer j23j2eh323"},
				},
			}),
			miruken.WithOptions(ServerOptions{
				Timeout: 100,
				Headers: []Header{
					{"Content-Encoding", "compress"},
				},
			}))
		var options ServerOptions
		suite.True(miruken.GetOptions(handler, &options))
		suite.Equal("https://netflix.com", options.Url)
		suite.Equal(100, options.Timeout)
		suite.ElementsMatch([]Header{
			{"Content-Key", "application/json"},
			{"Authorization", "Bearer j23j2eh323"},
			{"Content-Encoding", "compress"}}, options.Headers)
	})

	suite.Run("NoMatch", func () {
		handler := miruken.NewRootHandler()
		var options ServerOptions
		suite.False(miruken.GetOptions(handler, &options))
		suite.Equal("", options.Url)
		suite.Equal(0, options.Timeout)
	})

	suite.Run("NoMatchCre", func () {
		handler := miruken.NewRootHandler()
		var options *ServerOptions
		suite.False(miruken.GetOptions(handler, &options))
		suite.Nil(options)
	})

	suite.Run("FromOptions", func () {
		handler := miruken.NewRootHandler(
			miruken.WithHandlerTypes(miruken.TypeOf[*FooOptionsHandler]()))
		foo     := new(Foo)
		result  := miruken.Build(handler, miruken.WithOptions(FooOptions{2})).
			Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(miruken.Handled, result)
		suite.Equal(2, foo.Count())
	})

	suite.Run("Errors", func () {
		suite.Run("Nil Options", func () {
			defer func() {
				suite.Equal("options cannot be nil", recover())
			}()
			miruken.WithOptions(nil)
		})

		suite.Run("Not Struct Options", func () {
			defer func() {
				suite.Equal("options must be a struct or *struct", recover())
			}()
			miruken.WithOptions(1)
		})
	})
}

func TestOptionsTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsTestSuite))
}