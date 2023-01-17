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

func (f *provider) Unmarshal(path string, flat bool, output any) error {
	return f.k.UnmarshalWithConf(path, output,
		koanf.UnmarshalConf{Tag: "path", FlatPaths: flat})
}

// P returns a config.Provider using the Koanf instance.
func P(k *koanf.Koanf) config.Provider {
	if k == nil {
		panic("k cannot be nil")
	}
	return &provider{k}
}
