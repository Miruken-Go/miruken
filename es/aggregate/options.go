package aggregate

import "github.com/miruken-go/miruken"

// Options customize aggregate operations.
type Options struct {
	Version miruken.Option[int]
}

// Version returns a miruken.Builder requesting a specific aggregate version.
func Version(version int) miruken.Builder {
	return miruken.Options(Options{Version: miruken.Set(version)})
}
