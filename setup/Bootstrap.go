package setup

import "github.com/miruken-go/miruken/promise"

type Bootstrap interface {
	Startup() (*promise.Promise[struct{}], error)
	Shutdown() (*promise.Promise[struct{}], error)
}
