package api

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
)

type (
	// Stash is a transient storage of data.
	Stash struct {
		root bool
		data map[any]any
	}

	// stashAction defines Stash operations.
	stashAction struct {
		key any
	}

	// stashGet gets a value from the Stash.
	stashGet struct {
		stashAction
		found bool
		val   any
	}

	// stashPut puts a value into the Stash.
	stashPut struct {
		stashAction
		val any
	}

	// stashDrop removes a value from the Stash.
	stashDrop struct {
		stashAction
	}
)

func (s *stashAction) CanFilter() bool {
	return false
}

func (g *stashGet) setValue(val any) {
	g.val   = val
	g.found = true
}

// NoConstructor prevents Stash from being created implicitly.
func (s *Stash) NoConstructor() {}

// Provide retrieves an item by key.
func (s *Stash) Provide(
	_*struct{
		provides.It; miruken.Strict
	  }, p *provides.It,
) any {
	return s.data[p.Key()]
}

// Get retrieves an item by key.
// Build is considered NotHandled if an item with the key is not found and
// this Stash is not rooted.  This allows retrieval to propagate up the chain.
func (s *Stash) Get(
	_ *handles.It, get *stashGet,
) miruken.HandleResult {
	if val, ok := s.data[get.key]; ok {
		get.setValue(val)
	} else if !s.root {
		return miruken.NotHandled
	}
	return miruken.Handled
}

// Put stores an item by key.
func (s *Stash) Put(
	_ *handles.It, put *stashPut,
) {
	s.data[put.key] = put.val
}

// Drop removes an item by key.
func (s *Stash) Drop(
	_ *handles.It, drop *stashDrop,
) {
	delete(s.data, drop.key)
}

func StashGetKey(
	handler miruken.Handler,
	key     any,
) (val any, ok bool) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(key) {
		panic("key cannot be nil")
	}
	get := &stashGet{}
	get.key = key
	if result := handler.Handle(get, false, nil); result.Handled() {
		val = get.val
		ok  = get.found
	}
	return
}

func StashGet[T any](
	handler miruken.Handler,
) (t T, ok bool) {
	if val, ok := StashGetKey(handler, miruken.TypeOf[T]()); ok {
		return val.(T), true
	}
	return
}

func StashPutKey(
	handler miruken.Handler,
	key     any,
	val     any,
) error {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(key) {
		panic("key cannot be nil")
	}
	put := &stashPut{val: val}
	put.key = key
	if result := handler.Handle(put, false, nil); result.IsError() {
		return result.Error()
	} else if !result.Handled() {
		return &miruken.NotHandledError{Callback: put}
	}
	return nil
}

func StashPut[T any](
	handler miruken.Handler,
	val     T,
) error {
	return StashPutKey(handler, miruken.TypeOf[T](), val)
}

func StashGetOrPutKey(
	handler miruken.Handler,
	key     any,
	val     any,
) (any, error) {
	if v, ok := StashGetKey(handler, key); !ok {
		return val, StashPutKey(handler, key, val)
	} else {
		return v, nil
	}
}

func StashGetOrPut[T any](
	handler miruken.Handler,
	val     T,
) (T, error) {
	if v, ok := StashGet[T](handler); !ok {
		return val, StashPut(handler, val)
	} else {
		return v, nil
	}
}

func StashGetOrPutKeyFunc(
	handler miruken.Handler,
	key     any,
	fun     func() (any, *promise.Promise[any]),
) (any, *promise.Promise[any], error) {
	if fun == nil {
		panic("fun cannot be nil")
	}
	if v, ok := StashGetKey(handler, key); !ok {
		if val, pv := fun(); pv != nil {
			return nil, promise.Then(pv, func(res any) any {
				if err := StashPutKey(handler, key, res); err != nil {
					panic(err)
				}
				return res
			}), nil
		} else {
			return val, nil, StashPutKey(handler, key, val)
		}
	} else {
		return v, nil, nil
	}
}

func StashGetOrPutFunc[T any](
	handler miruken.Handler,
	fun     func() (T, *promise.Promise[T]),
) (T, *promise.Promise[T], error) {
	if fun == nil {
		panic("fun cannot be nil")
	}
	if v, ok := StashGet[T](handler); !ok {
		if val, pv := fun(); pv != nil {
			return val, promise.Then(pv, func(res T) T {
				if err := StashPut(handler, res); err != nil {
					panic(err)
				}
				return res
			}), nil
		} else {
			return val, nil, StashPut(handler, val)
		}
	} else {
		return v, nil, nil
	}
}

func StashDropKey(
	handler miruken.Handler,
	key     any,
) (err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(key) {
		panic("key cannot be nil")
	}
	drop := &stashDrop{}
	drop.key = key
	if result := handler.Handle(drop, false, nil); result.IsError() {
		err = result.Error()
	} else if !result.Handled() {
		err = &miruken.NotHandledError{Callback: drop}
	}
	return
}

func StashDrop[T any](
	handler miruken.Handler,
) error {
	return StashDropKey(handler, miruken.TypeOf[T]())
}

// NewStash creates a new Stash.
// When root is true, retrieval will not fail if not found.
func NewStash(root bool) *Stash {
	return &Stash{
		root,
		make(map[any]any),
	}
}