package json

// SurrogateMapper maps concepts to values that are more suitable
// for transmission over a json api.  The replacement type usually
// implements api.Surrogate to allow infrastructure to obtain the
// original value using the `Original` method.
type SurrogateMapper struct {}


