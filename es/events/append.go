package events

type (
	Stream struct {
		events []any
	}
)

// Append is a fluent builder to for appending events to Stream.
func Append(events ...any) *Stream {
	return &Stream{events: events}
}
