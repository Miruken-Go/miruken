package api

import (
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/maps"
	"github.com/miruken-go/miruken/promise"
	"reflect"
	"strings"
	"sync"
)

type (
	// Message is an envelope for polymorphic payloads.
	Message struct {
		Payload any
	}

	// Surrogate replaces a value with another for api transmission.
	Surrogate interface {
		Original(composer miruken.Handler) (any, error)
	}

	// Polymorphism is an enum that defines how type
	// discriminators are included in polymorphic messages.
	Polymorphism uint8

	// Options provide options for controlling api messaging.
	Options struct {
		Polymorphism   miruken.Option[Polymorphism]
		TypeInfoFormat string
		TypeFieldValue string
	}

	// TypeFieldInfo defines metadata for polymorphic serialization.
	TypeFieldInfo struct {
		TypeField   string
		TypeValue   string
		ValuesField string
	}

	// GoPolymorphismMapper provides polymorphic support for GO types.
	GoPolymorphismMapper struct{}
)

const (
	PolymorphismNone Polymorphism = 0
	PolymorphismRoot Polymorphism = 1 << iota
)

// GoPolymorphismMapper

// TypeInfo uses package and name to generate type metadata.
func (m *GoPolymorphismMapper) TypeInfo(
	_ *struct {
		maps.It
		maps.Format `to:"type:info"`
	}, maps *maps.It,
) (TypeFieldInfo, error) {
	val := reflect.TypeOf(maps.Source()).String()
	if strings.HasPrefix(val, "*") {
		val = val[1:]
	}
	if strings.HasPrefix(val, "[]*") {
		val = "[]" + val[3:]
	}
	return TypeFieldInfo{
		TypeField:   "@type",
		TypeValue:   val,
		ValuesField: "@values",
	}, nil
}

func (m *GoPolymorphismMapper) Static(
	_ *struct {
		miruken.Strict
		b     creates.It `key:"bool"`
		i     creates.It `key:"int"`
		i8    creates.It `key:"int8"`
		i16   creates.It `key:"int16"`
		i32   creates.It `key:"int32"`
		i64   creates.It `key:"int64"`
		ui    creates.It `key:"uint"`
		ui8   creates.It `key:"uint8"`
		ui16  creates.It `key:"uint16"`
		ui32  creates.It `key:"uint32"`
		ui64  creates.It `key:"uint64"`
		f32   creates.It `key:"float32"`
		f64   creates.It `key:"float64"`
		st    creates.It `key:"string"`
		a     creates.It `key:"interface {}"`
		bs    creates.It `key:"[]bool"`
		is    creates.It `key:"[]int"`
		i8s   creates.It `key:"[]int8"`
		i16s  creates.It `key:"[]int16"`
		i32s  creates.It `key:"[]int32"`
		i64s  creates.It `key:"[]int64"`
		uis   creates.It `key:"[]uint"`
		ui8s  creates.It `key:"[]uint8"`
		ui16s creates.It `key:"[]uint16"`
		ui32s creates.It `key:"[]uint32"`
		ui64s creates.It `key:"[]uint64"`
		f32s  creates.It `key:"[]float32"`
		f64s  creates.It `key:"[]float64"`
		sts   creates.It `key:"[]string"`
		as    creates.It `key:"[]interface {}"`
	}, create *creates.It,
) any {
	if key, ok := create.Key().(string); ok {
		if proto, ok := staticTypeMap[key]; ok {
			return proto
		}
	}
	return nil
}

func (m *GoPolymorphismMapper) Dynamic(
	_ *struct {
		miruken.Strict
		creates.It
	}, create *creates.It,
	ctx miruken.HandleContext,
) any {
	if key, ok := create.Key().(string); ok {
		dynamicLock.RLock()
		proto := dynamicTypeMap[key]
		dynamicLock.RUnlock()
		if proto == nil {
			if strings.HasPrefix(key, "*") {
				if id, _, err := miruken.CreateKey[any](ctx.Composer(), key[1:]); err == nil {
					proto = id
				}
			} else if strings.HasPrefix(key, "[]") {
				if el, _, err := miruken.CreateKey[any](ctx.Composer(), key[2:]); err == nil {
					proto = reflect.New(reflect.SliceOf(reflect.TypeOf(el))).Interface()
				}
			}
		}
		if proto != nil {
			dynamicLock.Lock()
			defer dynamicLock.Unlock()
			if p := dynamicTypeMap[key]; p == nil {
				dynamicTypeMap[key] = proto
			}
		}
		return proto
	}
	return nil
}

// UnknownTypeIdError reports an invalid type discriminator.
type UnknownTypeIdError struct {
	TypeId string
	Reason error
}

func (e *UnknownTypeIdError) Error() string {
	return fmt.Sprintf("unknown type id '%s': %s", e.TypeId, e.Reason.Error())
}

func (e *UnknownTypeIdError) Unwrap() error {
	return e.Reason
}

// MalformedErrorError reports an invalid error payload.
type MalformedErrorError struct {
	Culprit any
}

func (e *MalformedErrorError) Error() string {
	return fmt.Sprintf("malformed error: %T", e.Culprit)
}

// Post sends a message without an expected response.
// A new Stash is created to manage any transit state.
// Returns an empty promise if the call is asynchronous.
func Post(
	handler miruken.Handler,
	message any,
) (p *promise.Promise[miruken.Void], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("post: panic: %v", r)
			}
		}
	}()
	return miruken.Command(stash, message)
}

// Send sends a request with an expected response.
// A new Stash is created to manage any transit state.
// Returns the TResponse if the call is synchronous or
// a promise of TResponse if the call is asynchronous.
func Send[TResponse any](
	handler miruken.Handler,
	request any,
) (r TResponse, pr *promise.Promise[TResponse], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(request) {
		panic("request cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("send: panic: %v", r)
			}
		}
	}()
	return miruken.Execute[TResponse](stash, request)
}

// Publish sends a message to all recipients.
// A new Stash is created to manage any transit state.
// Returns an empty promise if the call is asynchronous.
func Publish(
	handler miruken.Handler,
	message any,
) (p *promise.Promise[miruken.Void], err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(message) {
		panic("message cannot be nil")
	}
	stash := miruken.AddHandlers(handler, NewStash(false))
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("publish: panic: %v", r)
			}
		}
	}()
	if pv, err := miruken.CommandAll(stash, message); err == nil {
		return pv, err
	} else if _, ok := err.(*miruken.NotHandledError); ok {
		return nil, nil
	} else {
		return pv, err
	}
}

var (
	// Polymorphic request encoding to include type information.
	Polymorphic = miruken.Options(Options{
		Polymorphism: miruken.Set(PolymorphismRoot),
	})

	// ToTypeInfo requests the type discriminator for a type.
	ToTypeInfo = maps.To("type:info")

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

	dynamicLock    sync.RWMutex
	dynamicTypeMap = make(map[string]any)
)
