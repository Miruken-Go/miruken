package openapi

// Config defines the configuration for openapi/swagger support.
type Config struct {
	AuthorizationUrl string
	TokenUrl         string
	ClientId         string
	Scopes           []struct {
		Name        string
		Description string
	}
	OpenIdConnectUrl string
}
