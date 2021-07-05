package miruken

import (
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
	_ Handles,
	foo     *Foo,
	options *struct{ _ FromOptions; Value FooOptions `bind:""`},
) {
	for i := 0; i < options.Value.Increment; i++ {
		foo.Inc()
	}
}

type OptionsTestSuite struct {
	suite.Suite
}

func (suite *OptionsTestSuite) TestOptions() {
	type Header struct{
		key   string
		value interface{}
	}

	type ServerOptions struct {
		Url     string
		Timeout int
		Headers []Header
	}

	suite.Run("Inline", func () {
		handler := NewRootHandler(WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
			Headers: []Header{
				{"Content-Type", "application/json"},
				{"Content-Encoding", "compress"},
			},
		}))
		var options ServerOptions
		suite.True(GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
		suite.ElementsMatch([]Header{
			{"Content-Type", "application/json"},
			{"Content-Encoding", "compress"}}, options.Headers)
	})

	suite.Run("InlineWithOptionsPtr", func () {
		serverOpt := new(ServerOptions)
		serverOpt.Url     = "https://playsoccer.com"
		serverOpt.Timeout = 30
		handler := NewRootHandler(WithOptions(serverOpt))
		var options ServerOptions
		suite.True(GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("Creates", func () {
		handler := NewRootHandler(WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
		}))
		var options *ServerOptions
		suite.True( GetOptions(handler, &options))
		suite.NotNil(options)
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("MergesInline", func () {
		handler := NewRootHandler(WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
		}))
		options := ServerOptions{Timeout: 60}
		suite.True(GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(60, options.Timeout)
	})

	suite.Run("MergesCreate", func () {
		handler := NewRootHandler(WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
		}))
		options := new (ServerOptions)
		options.Url = "https://improving.com"
		suite.True(GetOptions(handler, options))
		suite.NotNil(options)
		suite.Equal("https://improving.com", options.Url)
		suite.Equal(30, options.Timeout)
	})

	suite.Run("Combines", func () {
		handler := NewRootHandler(
			WithOptions(ServerOptions{
				Url:"https://directv.com",
			}),
			WithOptions(ServerOptions{
				Timeout: 60,
			}))
		var options ServerOptions
		suite.True(GetOptions(handler, &options))
		suite.Equal("https://directv.com", options.Url)
		suite.Equal(60, options.Timeout)
	})

	suite.Run("AppendsSlice", func () {
		handler := NewRootHandler(
			WithOptions(ServerOptions{
				Url:"https://netflix.com",
				Headers: []Header{
					{"Content-Type", "application/json"},
					{"Authorization", "Bearer j23j2eh323"},
				},
			}),
			WithOptions(ServerOptions{
				Timeout: 100,
				Headers: []Header{
					{"Content-Encoding", "compress"},
				},
			}))
		var options ServerOptions
		suite.True(GetOptions(handler, &options))
		suite.Equal("https://netflix.com", options.Url)
		suite.Equal(100, options.Timeout)
		suite.ElementsMatch([]Header{
			{"Content-Type", "application/json"},
			{"Authorization", "Bearer j23j2eh323"},
			{"Content-Encoding", "compress"}}, options.Headers)
	})

	suite.Run("NoMatch", func () {
		handler := NewRootHandler()
		var options ServerOptions
		suite.False(GetOptions(handler, &options))
		suite.Equal("", options.Url)
		suite.Equal(0, options.Timeout)
	})

	suite.Run("NoMatchCre", func () {
		handler := NewRootHandler()
		var options *ServerOptions
		suite.False(GetOptions(handler, &options))
		suite.Nil(options)
	})

	suite.Run("FromOptions", func () {
		handler := NewRootHandler(WithHandlers(new(FooOptionsHandler)))
		foo     := new(Foo)
		result  := Build(handler, WithOptions(FooOptions{2})).
			Handle(foo, false, nil)
		suite.False(result.IsError())
		suite.Equal(Handled, result)
		suite.Equal(2, foo.Count())
	})

	suite.Run("Errors", func () {
		suite.Run("Nil Options", func () {
			defer func() {
				suite.Equal("options cannot be nil", recover())
			}()
			WithOptions(nil)
		})

		suite.Run("Not Struct Options", func () {
			defer func() {
				suite.Equal("options must be a struct or *struct", recover())
			}()
			WithOptions(1)
		})
	})
}

func TestOptionsTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsTestSuite))
}