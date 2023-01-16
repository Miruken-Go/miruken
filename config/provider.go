package config

// Provider defines the api for configuration providers
// to implement to expose configuration information.
// output can be a pointer to a struct or map[string]any
type Provider interface {
	Unmarshal(path string, output any) error
}
