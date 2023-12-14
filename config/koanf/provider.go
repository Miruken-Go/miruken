package koanf

import (
	"strconv"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
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

// Merge extends the default merge to include slice conversions.
func Merge(src, dest map[string]any) error {
	ConvertSlices(src)
	maps.Merge(src, dest)
	return nil
}

// MergeStrict extends the strict merge to include slice conversions.
func MergeStrict(src, dest map[string]any) error {
	ConvertSlices(src)
	return maps.MergeStrict(src, dest)
}

// ConvertSlices converts maps with all integral keys into a
// slice with corresponding indices.
// returns the slice and true if successful
func ConvertSlices(m map[string]any) (any, bool) {
	var (
		invalid bool
		slice   []any
	)
	for k, v := range m {
		if c, ok := v.(map[string]any); ok {
			if cs, ok := ConvertSlices(c); ok {
				v, m[k] = cs, cs
			}
		}
		if !invalid {
			if i, err := strconv.Atoi(k); err == nil {
				if slice == nil {
					slice = make([]any, len(m))
				}
				if i >= len(slice) {
					ns := make([]any, i+1)
					copy(ns, slice)
					slice = ns
				}
				slice[i] = v
			} else if slice != nil {
				invalid = true
			}
		}
	}
	if slice != nil && !invalid {
		return slice, true
	}
	return nil, false
}
