package seq

func Not[V any](f func(V) bool) func(V) bool {
	return func(v V) bool {
		return !f(v)
	}
}