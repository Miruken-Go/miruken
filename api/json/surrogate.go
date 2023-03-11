package json

import "github.com/miruken-go/miruken/creates"

// SurrogateMapper maps concepts to values that are more suitable
// for transmission over a json api.  The replacement type usually
// implements api.Surrogate to allow infrastructure to obtain the
// original value using the `Original` method.
type SurrogateMapper struct {}

func (m *SurrogateMapper) New(
	_*struct{
		o creates.It `key:"json.OutcomeSurrogate"`
		e creates.It `key:"json.ErrorSurrogate"`
		c creates.It `key:"json.ConcurrentSurrogate"`
	    s creates.It `key:"json.SequentialSurrogate"`
	  }, create *creates.It,
) any {
	switch create.Key() {
	case "json.OutcomeSurrogate":
		return new(OutcomeSurrogate)
	case "json.ErrorSurrogate":
		return new(ErrorSurrogate)
	case "json.ConcurrentSurrogate":
		return new(ConcurrentSurrogate)
	case "json.SequentialSurrogate":
		return new(SequentialSurrogate)
	}
	return nil
}

