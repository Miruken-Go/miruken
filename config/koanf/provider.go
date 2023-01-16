package koanf

import (
	"github.com/knadh/koanf"
)

// provider of configurations populated by the koanf library.
// https://github.com/knadh/koanf
type provider struct {
	k *koanf.Koanf
}

func (f *provider) Unmarshal(path string, output any) error {
	return f.k.Unmarshal("", output)
}

// Use returns a config.Provider using the Koanf instance.
func Use(k *koanf.Koanf) any {
	if k == nil {
		panic("k cannot be nil")
	}
	return &provider{k}
}
