package test

import (
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
)

// Benchmark fixtures

// BenchProvider provides Foo and Bar instances.
type BenchProvider struct{}

func (p *BenchProvider) ProvideFoo(*provides.It) *Foo {
	return &Foo{Counted{1}}
}

func (p *BenchProvider) ProvideBar(*provides.It) *Bar {
	return &Bar{Counted{1}}
}

// BenchSingletonProvider provides singleton Foo and Bar.
type BenchSingletonProvider struct{}

func (p *BenchSingletonProvider) ProvideFoo(
	_ *struct {
		provides.It
		provides.Single
	},
) *Foo {
	return &Foo{Counted{1}}
}

func (p *BenchSingletonProvider) ProvideBar(
	_ *struct {
		provides.It
		provides.Single
	},
) *Bar {
	return &Bar{Counted{1}}
}

// Benchmarks

func BenchmarkResolve(b *testing.B) {
	b.Run("Transient", func(b *testing.B) {
		handler, _ := setup.New().Specs(&BenchProvider{}).Context()
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			miruken.Resolve[*Foo](handler)
		}
	})

	b.Run("Singleton", func(b *testing.B) {
		handler, _ := setup.New().Specs(&BenchSingletonProvider{}).Context()
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			miruken.Resolve[*Foo](handler)
		}
	})

	b.Run("SingletonRepeated", func(b *testing.B) {
		handler, _ := setup.New().Specs(&BenchSingletonProvider{}).Context()
		// Prime the cache
		miruken.Resolve[*Foo](handler)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			miruken.Resolve[*Foo](handler)
		}
	})
}

func BenchmarkResolveAll(b *testing.B) {
	handler, _ := setup.New().Specs(
		&BenchProvider{},
		&FooProvider{},
	).Context()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		miruken.ResolveAll[*Foo](handler)
	}
}

func BenchmarkResolveWithDependency(b *testing.B) {
	handler, _ := setup.New().Specs(
		&BenchDepHandler{},
		&BenchProvider{},
	).Context()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		miruken.Execute[any](handler, new(Foo))
	}
}
