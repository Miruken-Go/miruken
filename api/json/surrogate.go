package json

import "github.com/miruken-go/miruken/creates"

// SurrogateMapper maps concepts to values that are more suitable
// for transmission over a json api.  The replacement type usually
// implements api.Surrogate to allow infrastructure to obtain the
// original value using the Original() method.
type SurrogateMapper struct{}

func (m *SurrogateMapper) New(
	_ *struct {
		_ creates.It `key:"json.Outcome"`
		_ creates.It `key:"json.Error"`
		_ creates.It `key:"json.Concurrent"`
		_ creates.It `key:"json.Sequential"`
	  }, create *creates.It,
) any {
	switch create.Key() {
	case "json.Outcome":
		return new(Outcome)
	case "json.Error":
		return new(Error)
	case "json.Concurrent":
		return new(Concurrent)
	case "json.Sequential":
		return new(Sequential)
	}
	return nil
}
