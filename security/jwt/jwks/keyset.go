package jwks

import (
	"encoding/json"
	"maps"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MicahParks/keyfunc/v2"
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

	return promise.New(func(resolve func(jwt.Keyfunc), reject func(error)) {
		jwks, err := keyfunc.Get(jwksURI, getOptions)
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
			atc := maps.Clone(*at)
			at = &atc
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
	jwks, err := keyfunc.NewJSON(jwksJSON)
	if err != nil {
		return nil, err
	}
	return jwks.Keyfunc, nil
}

var getOptions = keyfunc.Options{
	RefreshRateLimit:  time.Minute * 5,
	RefreshTimeout:    time.Second * 10,
	RefreshUnknownKID: true,
}
