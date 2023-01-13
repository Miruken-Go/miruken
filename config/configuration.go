package config

type (
	// Configuration marks a type as a configuration and
	// provides an opportunity to initialize after configured.
	Configuration interface {
		ConfigurationReady()
	}
)
