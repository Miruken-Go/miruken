package test

import (
	"testing"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/setup"
)

// Benchmark fixtures

// BenchHandler handles Foo callbacks with no dependencies.
type BenchHandler struct{}

func (h *BenchHandler) HandleFoo(
	_ *handles.It, foo *Foo,
) {
	foo.Inc()
}

// BenchDepHandler handles Foo callbacks with a dependency.
type BenchDepHandler struct{}

func (h *BenchDepHandler) HandleFoo(
	_ *handles.It, foo *Foo,
	bar *Bar,
) {
	foo.Inc()
}

// BenchMultiReturnHandler returns a value.
type BenchMultiReturnHandler struct{}

func (h *BenchMultiReturnHandler) HandleFoo(
	_ *handles.It, foo *Foo,
) *Foo {
	foo.Inc()
	return foo
}

// BenchInterfaceHandler handles Counter interface.
type BenchInterfaceHandler struct{}

func (h *BenchInterfaceHandler) HandleCounter(
	_ *handles.It, counter Counter,
) {
	counter.Inc()
}

// Benchmarks

func BenchmarkHandleDispatch(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		handler, _ := setup.New().Specs(&BenchHandler{}).Context()
		foo := new(Foo)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			handler.Handle(foo, false, nil)
		}
	})

	b.Run("WithReturn", func(b *testing.B) {
		handler, _ := setup.New().Specs(&BenchMultiReturnHandler{}).Context()
		foo := new(Foo)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			handler.Handle(foo, false, nil)
		}
	})

	b.Run("Interface", func(b *testing.B) {
		handler, _ := setup.New().Specs(&BenchInterfaceHandler{}).Context()
		foo := new(Foo)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			handler.Handle(foo, false, nil)
		}
	})

	b.Run("WithDependency", func(b *testing.B) {
		handler, _ := setup.New().Specs(
			&BenchDepHandler{},
			&BenchProvider{},
		).Context()
		foo := new(Foo)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			handler.Handle(foo, false, nil)
		}
	})
}

func BenchmarkExecute(b *testing.B) {
	handler, _ := setup.New().Specs(&BenchMultiReturnHandler{}).Context()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		miruken.Execute[*Foo](handler, new(Foo))
	}
}

func BenchmarkGreedyDispatch(b *testing.B) {
	handler, _ := setup.New().Specs(
		&BenchHandler{},
		&EverythingHandler{},
	).Context()
	foo := new(Foo)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		handler.Handle(foo, true, nil)
	}
}
