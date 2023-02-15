package jsonstd

import "github.com/miruken-go/miruken/creates"

// SurrogateMapper maps concepts to values that are more suitable
// for transmission over a standard polymorphic json api.
type SurrogateMapper struct {}

func (m *SurrogateMapper) New(
	_*struct{
		sr creates.It `key:"jsonstd.ScheduledResultSurrogate"`
	  }, create *creates.It,
) any {
	switch create.Key() {
	case "jsonstd.ScheduledResultSurrogate":
		return new(ScheduledResultSurrogate)
	}
	return nil
}