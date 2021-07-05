package miruken

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type OptionsTestSuite struct {
	suite.Suite
}

func (suite *OptionsTestSuite) TestOptions() {
	type ServerOptions struct {
		Url     string
		Timeout int
	}

	suite.Run("Inline", func () {
		handler := NewRootHandler(WithOptions(ServerOptions{
			Url:     "https://playsoccer.com",
			Timeout: 30,
		}))
		var options ServerOptions
		suite.True(GetOptions(handler, &options))
		suite.Equal("https://playsoccer.com", options.Url)
		suite.Equal(30, options.Timeout)
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