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

func (c *Config) ScopeNames() []string {
	ret := make([]string, len(c.Scopes))
	for i, s := range c.Scopes {
		ret[i] = s.Name
	}
	return ret
}

func (c *Config) ScopeMap() map[string]string {
	ret := map[string]string{}
	for _, s := range c.Scopes {
		ret[s.Name] = s.Description
	}
	return ret
}
