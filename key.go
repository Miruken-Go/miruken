package miruken

import (
	"fmt"
	"github.com/miruken-go/miruken/promise"
	"reflect"
)

// Key is a miruken.DependencyResolver for resolving
// values referenced by a string key.
type Key string


//goland:noinspection GoMixedReceiverTypes
func (k *Key) InitWithTag(tag reflect.StructTag) error {
	if key, ok := tag.Lookup("of"); ok {
		*k = Key(key)
	}
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (k Key) Resolve(
	typ reflect.Type,
	dep DependencyArg,
	ctx HandleContext,
) (v reflect.Value, pv *promise.Promise[reflect.Value], err error) {
	var builder ProvidesBuilder
	provides := builder.WithKey(string(k)).New()
	if result, pr, err2 := provides.Resolve(ctx, false); err2 != nil {
		err = fmt.Errorf("key: unable to resolve dependency %q: %w", k, err2)
	} else if pr == nil {
		if result != nil {
			v = reflect.ValueOf(result)
		} else if dep.Optional() {
			v = reflect.Zero(typ)
		} else {
			err = fmt.Errorf("key: unable to resolve dependency %q", k)
		}
	} else {
		pv = promise.Then(pr, func(res any) reflect.Value {
			var val reflect.Value
			if res != nil {
				val = reflect.ValueOf(res)
			} else if dep.Optional() {
				val = reflect.Zero(typ)
			} else {
				panic(fmt.Errorf("key: unable to resolve dependency %q", k))
			}
			return val
		})
	}
	return
}

