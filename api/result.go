package api

import "github.com/miruken-go/miruken/either"

// Failure returns a new failed result.
func Failure(val error) either.Monad[error, any] {
	return either.Left(val)
}

// Success returns a new successful result.
func Success[R any](val R) either.Monad[error, R] {
	return either.Right(val)
}
