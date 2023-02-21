package api

import (
	"github.com/miruken-go/miruken/maps"
	"mime"
)

var (
	ToJson   = maps.To("application/json", nil)
	FromJson = maps.From("application/json", nil)
)


// ParseContentType parses the contentType into a
// maps.Format suitable for mapping in the direction.
func ParseContentType(
	contentType string,
	direction   maps.Direction,
) (*maps.Format, error) {
	if mt, params, err := mime.ParseMediaType(contentType); err != nil {
		return nil, err
	} else if direction == maps.DirectionTo {
		return maps.To(mt, params), nil
	} else {
		return maps.From(mt, params), nil
	}
}
