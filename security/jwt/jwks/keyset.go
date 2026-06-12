package jwks

import (
	"context"
	"encoding/json"
	"maps"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/miruken-go/miruken/promise"
)

type (
	// KeySet manages JWKS (Json Web Key Set) using keyfunc module.
	KeySet struct {
		at   atomic.Pointer[map[string]jwt.Keyfunc]
		lock sync.Mutex
	}
)

func (f *KeySet) At(
	jwksURI string,
) *promise.Promise[jwt.Keyfunc] {
	if jwksURI == "" {
		panic("jwksURI cannot be empty")
	}

	if at := f.at.Load(); at != nil {
		if fn, ok := (*at)[jwksURI]; ok {
			return promise.Resolve(fn)
		}
	}

	return promise.New(nil, func(resolve func(jwt.Keyfunc), reject func(error), onCancel func(func())) {
		jwks, err := keyfunc.NewDefaultOverrideCtx(
			context.Background(), []string{jwksURI}, refreshOverride)
		if err != nil {
			reject(err)
			return
		}
		f.lock.Lock()
		defer f.lock.Unlock()
		at := f.at.Load()
		if at != nil {
			if fn, ok := (*at)[jwksURI]; ok {
				resolve(fn)
				return
			}
			at = new(maps.Clone(*at))
		} else {
			at = &map[string]jwt.Keyfunc{jwksURI: jwks.Keyfunc}
		}
		f.at.Store(at)
		resolve(jwks.Keyfunc)
	})
}

func (f *KeySet) From(
	jwksJSON json.RawMessage,
) (jwt.Keyfunc, error) {
	jwks, err := keyfunc.NewJWKSetJSON(jwksJSON)
	if err != nil {
		return nil, err
	}
	return jwks.Keyfunc, nil
}

// refreshOverride preserves the original JWKS refresh tuning under keyfunc v3.
// A 10s HTTP timeout is set explicitly; refresh-on-unknown-KID rate-limited to
// 5 minutes is the keyfunc v3 default.
var refreshOverride = keyfunc.Override{
	HTTPTimeout: time.Second * 10,
}
