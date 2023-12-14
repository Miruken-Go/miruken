package api

import (
	"fmt"
	maps2 "maps"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/maps"
)

type (
	// Polymorphism enumerates how type information should be included in messages.
	Polymorphism uint8

	// TypeFieldInfo defines the type information for polymorphic messages.
	TypeFieldInfo struct {
		TypeField   string
		TypeValue   string
		ValuesField string
	}

	// GoPolymorphism provides type information using GO types names.
	GoPolymorphism struct{}

	// UnknownTypeIdError reports an invalid type discriminator.
	UnknownTypeIdError struct {
		TypeId string
		Cause  error
	}

	// Late is a container for late polymorphic results.
	// It is used be serializers to explicitly request polymorphic
	// behavior and avoid circular calls with any type.
	Late struct {
		Value any
	}
)

const (
	PolymorphismNone Polymorphism = 0
	PolymorphismRoot Polymorphism = 1 << iota
)

var (
	// Polymorphic instructs type information to be included.
	Polymorphic = miruken.Options(Options{
		Polymorphism: miruken.Set(PolymorphismRoot),
	})

	// NoPolymorphism instructs type information to be suppressed.
	NoPolymorphism = miruken.Options(Options{
		Polymorphism: miruken.Set(PolymorphismNone),
	})

	// ToTypeInfo formats a value into corresponding type information.
	ToTypeInfo = maps.To("type:info", nil)
)

// TypeInfo uses package and name to generate type metadata.
func (m *GoPolymorphism) TypeInfo(
	_ *struct {
		maps.Format `to:"type:info"`
	}, it *maps.It,
) (TypeFieldInfo, error) {
	val := reflect.TypeOf(it.Source()).String()
	val = strings.TrimPrefix(val, "*")
	if strings.HasPrefix(val, "[]*") {
		val = "[]" + val[3:]
	}
	return TypeFieldInfo{
		TypeField:   "@type",
		TypeValue:   val,
		ValuesField: "@values",
	}, nil
}

func (m *GoPolymorphism) Static(
	_ *struct {
		creates.Strict
		_ creates.It `key:"bool"`
		_ creates.It `key:"int"`
		_ creates.It `key:"int8"`
		_ creates.It `key:"int16"`
		_ creates.It `key:"int32"`
		_ creates.It `key:"int64"`
		_ creates.It `key:"uint"`
		_ creates.It `key:"uint8"`
		_ creates.It `key:"uint16"`
		_ creates.It `key:"uint32"`
		_ creates.It `key:"uint64"`
		_ creates.It `key:"float32"`
		_ creates.It `key:"float64"`
		_ creates.It `key:"string"`
		_ creates.It `key:"interface {}"`
		_ creates.It `key:"[]bool"`
		_ creates.It `key:"[]int"`
		_ creates.It `key:"[]int8"`
		_ creates.It `key:"[]int16"`
		_ creates.It `key:"[]int32"`
		_ creates.It `key:"[]int64"`
		_ creates.It `key:"[]uint"`
		_ creates.It `key:"[]uint8"`
		_ creates.It `key:"[]uint16"`
		_ creates.It `key:"[]uint32"`
		_ creates.It `key:"[]uint64"`
		_ creates.It `key:"[]float32"`
		_ creates.It `key:"[]float64"`
		_ creates.It `key:"[]string"`
		_ creates.It `key:"[]interface {}"`
	}, create *creates.It,
) any {
	if key, ok := create.Key().(string); ok {
		if proto, ok := staticTypeMap[key]; ok {
			return proto
		}
	}
	return nil
}

func (m *GoPolymorphism) Dynamic(
	_ *struct{ creates.Strict }, create *creates.It,
	ctx miruken.HandleContext,
) any {
	if key, ok := create.Key().(string); ok {
		if types := dynamicTypeMap.Load(); types != nil {
			if proto, ok := (*types)[key]; ok {
				return proto
			}
		}
		var proto any
		if strings.HasPrefix(key, "*") {
			if id, _, err := creates.Key[any](ctx, key[1:]); err == nil {
				proto = id
			}
		} else if strings.HasPrefix(key, "[]") {
			if el, _, err := creates.Key[any](ctx, key[2:]); err == nil {
				proto = reflect.New(reflect.SliceOf(reflect.TypeOf(el))).Interface()
			}
		}
		if proto != nil {
			dynamicTypeLock.Lock()
			defer dynamicTypeLock.Unlock()
			types := dynamicTypeMap.Load()
			if types != nil {
				if proto, ok := (*types)[key]; ok {
					return proto
				}
				db := maps2.Clone(*types)
				types = &db
			} else {
				types = &map[string]any{}
			}
			(*types)[key] = proto
			dynamicTypeMap.Store(types)
			return proto
		}
	}
	return nil
}

func (e *UnknownTypeIdError) Error() string {
	return fmt.Sprintf("unknown type id %q: %s", e.TypeId, e.Cause.Error())
}

func (e *UnknownTypeIdError) Unwrap() error {
	return e.Cause
}

var (
	staticTypeMap = map[string]any{
		"bool":           new(bool),
		"int":            new(int),
		"int8":           new(int8),
		"int16":          new(int16),
		"int32":          new(int32),
		"int64":          new(int64),
		"uint":           new(uint),
		"uint8":          new(uint8),
		"uint16":         new(uint16),
		"uint32":         new(uint32),
		"uint64":         new(uint64),
		"float32":        new(float32),
		"float64":        new(float64),
		"string":         new(string),
		"interface {}":   new(any),
		"[]bool":         new([]bool),
		"[]int":          new([]int),
		"[]int8":         new([]int8),
		"[]int16":        new([]int16),
		"[]int32":        new([]int32),
		"[]int64":        new([]int64),
		"[]uint":         new([]uint),
		"[]uint8":        new([]uint8),
		"[]uint16":       new([]uint16),
		"[]uint32":       new([]uint32),
		"[]uint64":       new([]uint64),
		"[]float32":      new([]float32),
		"[]float64":      new([]float64),
		"[]string":       new([]string),
		"[]interface {}": new([]any),
	}

	dynamicTypeLock sync.Mutex
	dynamicTypeMap  = atomic.Pointer[map[string]any]{}
)
