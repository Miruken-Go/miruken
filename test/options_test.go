package test

import (
	"github.com/miruken-go/miruken"
	"github.com/stretchr/testify/suite"
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
	_*struct{ miruken.FromOptions }, options FooOptions,
) {
	for i := 0; i < options.Increment; i++ {
		foo.Inc()
	}
}

type OptionsTestSuite struct {
	suite.Suite
}

func (suite *OptionsTestSuite) Setup() (miruken.Handler, error) {
	return miruken.Setup()
}

func (suite *OptionsTestSuite) SetupWith(specs ... any) (miruken.Handler, error) {
	return miruken.Setup(miruken.HandlerSpecs(specs...))
}

func (suite *OptionsTestSuite) TestOptions() {
	type Header struct{
		key   string
		value any
	}

	type ServerOptions struct {
		Url       string
		Timeout   int
		KeepAlive miruken.Option[bool]
		Headers   []Header
	}

	suite.Run("Inline", func () {
		handler, _ :=suite.Setup()
		handler = miruken.BuildUp(handler,
			miruken.Options(ServerOptions{
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
		handler, _ := suite.Setup()
		handler = miruken.BuildUp(handler, miruken.Options(serverOpt))
		var options ServerOptions
		suite.True(miruken.GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("Creates", func () {
		handler, _ := suite.Setup()
		handler = miruken.BuildUp(handler, miruken.Options(ServerOptions{
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
		handler, _ := suite.Setup()
		handler = miruken.BuildUp(handler, miruken.Options(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
		}))
		options := ServerOptions{Timeout: 60}
		suite.True(miruken.GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(60, options.Timeout)
	})

	suite.Run("MergesCreate", func () {
		handler, _ := suite.Setup()
		handler = miruken.BuildUp(handler, miruken.Options(ServerOptions{
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
		handler, _ := suite.Setup()
		handler = miruken.BuildUp(handler,
			miruken.Options(ServerOptions{
				Url:       "https://directv.com",
				KeepAlive: miruken.Set(true),
			}),
			miruken.Options(ServerOptions{
				Timeout:   60,
				KeepAlive: miruken.Set(false),
			}))
		var options ServerOptions
		suite.True(miruken.GetOptions(handler, &options))
		suite.Equal("https://directv.com", options.Url)
		suite.Equal(60, options.Timeout)
		suite.Equal(miruken.Set(false), options.KeepAlive)
	})

	suite.Run("AppendsSlice", func () {
		handler, _ := suite.Setup()
		handler = miruken.BuildUp(handler,
			miruken.Options(ServerOptions{
				Url:"https://netflix.com",
				Headers: []Header{
					{"Content-Key", "application/json"},
					{"Authorization", "Bearer j23j2eh323"},
				},
			}),
			miruken.Options(ServerOptions{
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
		handler, _ := suite.Setup()
		var options ServerOptions
		suite.False(miruken.GetOptions(handler, &options))
		suite.Equal("", options.Url)
		suite.Equal(0, options.Timeout)
	})

	suite.Run("NoMatchCre", func () {
		handler, _ := suite.Setup()
		var options *ServerOptions
		suite.False(miruken.GetOptions(handler, &options))
		suite.Nil(options)
	})

	suite.Run("FromOptions", func () {
		handler, _ := suite.SetupWith(&FooOptionsHandler{})
		foo    := new(Foo)
		result := miruken.BuildUp(handler, miruken.Options(FooOptions{2})).
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
			miruken.Options(nil)
		})

		suite.Run("Not Struct Options", func () {
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