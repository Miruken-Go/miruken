package events

type (
	// Stream represents a collection of events to be applied to an Aggregate.
	Stream []any
)


// Append is a fluent builder to for appending events to a Stream.
func Append(events ...any) Stream {
	return events
}
