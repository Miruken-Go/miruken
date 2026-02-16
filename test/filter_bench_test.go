package test

import (
	"testing"

	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/setup"
)

// Benchmark fixtures

// BenchFilterHandler handles Foo with a filter pipeline.
type BenchFilterHandler struct{}

func (h *BenchFilterHandler) HandleFoo(
	_ *struct {
		handles.It
		NullFilter
	}, foo *Foo,
) {
	foo.Inc()
}

// Benchmarks

func BenchmarkFilterPipeline(b *testing.B) {
	b.Run("NoFilters", func(b *testing.B) {
		handler, _ := setup.New().Specs(&BenchHandler{}).Context()
		foo := new(Foo)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			handler.Handle(foo, false, nil)
		}
	})

	b.Run("OneFilter", func(b *testing.B) {
		handler, _ := setup.New().Specs(
			&BenchFilterHandler{},
			&NullFilter{},
		).Context()
		foo := new(Foo)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			handler.Handle(foo, false, nil)
		}
	})

	b.Run("MultipleFilters", func(b *testing.B) {
		handler, _ := setup.New().Specs(
			&FilteringHandler{},
			&ConsoleLogger{},
			&LogFilter{},
			&ExceptionFilter{},
			&NullFilter{},
		).Context()
		bar := new(BarC)
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			handler.Handle(bar, false, nil)
		}
	})
}
