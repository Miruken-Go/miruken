package jwks

import (
	"encoding/json"
	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"sync"
	"time"
)

type (
	// KeySet manages JWKS (Json Web Key Set) using keyfunc module.
	KeySet struct {
		at   map[string]jwt.Keyfunc
		lock sync.RWMutex
	}
)


func (f *KeySet) Constructor(
	_*struct{
		provides.It
		provides.Single
	  },
) {
	f.at = make( map[string]jwt.Keyfunc)
}

func (f *KeySet) At(
	jwksURI string,
) *promise.Promise[jwt.Keyfunc] {
	if jwksURI == "" {
		panic("jwksURI cannot be empty")
	}
	f.lock.RLock()
	if fn, ok := f.at[jwksURI]; ok {
		return promise.Resolve(fn)
	}
	f.lock.RUnlock()

	return promise.New(func(resolve func(jwt.Keyfunc), reject func(error)) {
		jwks, err := keyfunc.Get(jwksURI, getOptions)
		if err != nil {
			reject(err)
		} else {
			f.lock.Lock()
			defer f.lock.Unlock()
			if fn, ok := f.at[jwksURI]; ok {
				resolve(fn)
			} else {
				f.at[jwksURI] = jwks.Keyfunc
				resolve(jwks.Keyfunc)
			}
		}
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
