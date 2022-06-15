package api

import "github.com/miruken-go/miruken"

type (
	// Stash is a temporary storage of data.
	Stash struct {
		root bool
		data map[any]any
	}

	// stashAction defines the Stash operations.
	stashAction struct {
		key any
	}

	// stashGet gets a value from the Stash.
	stashGet struct {
		stashAction
		val any
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

func (s *Stash) NoConstructor() {}

func (s *Stash) Provide(
	_*struct{
		miruken.Provides; miruken.Strict
	 }, provides *miruken.Provides,
) any {
	return s.data[provides.Key()]
}

func (s *Stash) Get(
	_*miruken.Handles, get *stashGet,
) miruken.HandleResult {
	if val, ok := s.data[get.key]; ok {
		get.val = val
		return miruken.Handled
	} else if s.root {
		return miruken.Handled
	}
	return miruken.NotHandled
}

func (s *Stash) Put(
	_*miruken.Handles, put *stashPut,
) {
	s.data[put.key] = put.val
}

func (s *Stash) Drop(
	_*miruken.Handles, drop *stashDrop,
) {
	delete(s.data, drop.key)
}

func StashGetKey(
	handler miruken.Handler,
	key     any,
) (val any, err error) {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	if miruken.IsNil(key) {
		panic("key cannot be nil")
	}
	get := &stashGet{}
	get.key = key
	if result := handler.Handle(get, false, nil); result.IsError() {
		err = result.Error()
	} else if result.IsHandled() {
		val = get.val
	} else {
		err = miruken.NotHandledError{}
	}
	return
}

func StashGet[T any](
	handler miruken.Handler,
) (t T, err error) {
	if val, e := StashGetKey(handler, miruken.TypeOf[T]()); e != nil {
		err = e
	} else if !miruken.IsNil(val) {
		t = val.(T)
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
	put := &stashGet{val: val}
	put.key = key
	if result := handler.Handle(put, false, nil); result.IsError() {
		return result.Error()
	} else if !result.IsHandled() {
		return miruken.NotHandledError{}
	}
	return nil
}

func StashPut[T any](
	handler miruken.Handler,
	val     T,
) error {
	return StashPutKey(handler, miruken.TypeOf[T](), val)
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
	} else if !result.IsHandled() {
		err = miruken.NotHandledError{}
	}
	return
}

func StashDrop[T any](
	handler miruken.Handler,
) error {
	return StashDropKey(handler, miruken.TypeOf[T]())
}

func NewStash(root bool) *Stash {
	return &Stash{
		root,
		make(map[any]any),
	}
}