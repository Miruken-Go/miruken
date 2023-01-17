package koanfp

import (
	"github.com/knadh/koanf"
	"github.com/miruken-go/miruken/config"
)

// provider of configurations populated by the koanf library.
// https://github.com/knadh/koanf
type provider struct {
	k *koanf.Koanf
}

func (f *provider) Unmarshal(path string, output any) error {
	return f.k.Unmarshal("", output)
}

// P returns a config.Provider using the Koanf instance.
func P(k *koanf.Koanf) config.Provider {
	if k == nil {
		panic("k cannot be nil")
	}
	return &provider{k}
}
