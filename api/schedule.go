package api

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/promise"
	"sync"
)

type (
	// ConcurrentBatch represents a batch of requests to execute concurrently.
	// The operation returns after all requests are completed and
	// includes all successes and failures.
	ConcurrentBatch struct {
		Requests []any
	}

	// SequentialBatch represents a batch of requests to execute sequentially.
	// The operation aborts after the first failure and returns the
	// successfully completed responses and first failure.
	SequentialBatch struct {
		Requests []any
	}

	// Published marks a message to be published to all consumers.
	Published struct {
		Message any
	}

	// ScheduledResult represents the results of a scheduled request.
	// The result is either an error (if fails) or success value.
	ScheduledResult struct {
		Responses []either.Either[error, any]
	}

	// Scheduler performs the scheduling of requests.
	Scheduler struct {}
)


// Scheduler

func (s *Scheduler) Constructor(
	_*struct{
	    miruken.Provides
	    miruken.Singleton
	  },
) {
}

func (s *Scheduler) HandleConcurrent(
	_ *miruken.Handles, concurrent ConcurrentBatch,
	composer miruken.Handler,
) *promise.Promise[ScheduledResult] {
	return promise.New(func(resolve func(ScheduledResult), reject func(error)) {
		requests := concurrent.Requests
		responses := make([]either.Either[error, any], len(requests))

		var waitGroup sync.WaitGroup
		waitGroup.Add(len(requests))

		for i, request := range requests {
			go func(idx int, req any) {
				defer waitGroup.Done()
				response, _ := process(req, composer)
				responses[idx] = response
			}(i, request)
		}

		waitGroup.Wait()
		resolve(ScheduledResult{responses})
	})
}

func (s *Scheduler) HandleSequential(
	_ *miruken.Handles, sequential SequentialBatch,
	composer miruken.Handler,
) *promise.Promise[ScheduledResult] {
	return promise.New(func(resolve func(ScheduledResult), reject func(error)) {
		requests := sequential.Requests
		var responses []either.Either[error, any]

		for _, request := range requests {
			response, success := process(request, composer)
			responses = append(responses, response)
			if !success {
				break
			}
		}

		resolve(ScheduledResult{responses})
	})
}

func (s *Scheduler) HandlePublish(
	_ *miruken.Handles, publish Published,
	composer miruken.Handler,
) (p *promise.Promise[miruken.Void], err error) {
	return Publish(composer, publish.Message)
}

func process(
	request any,
	handler miruken.Handler,
) (either.Either[error, any], bool) {
	res, pr, err := Send[any](handler, request)
	if err != nil {
		return Failure(err), false
	}
	if pr != nil {
		if res, err = pr.Await(); err != nil {
			return Failure(err), false
		}
	}
	return Success(res), true
}


// Sequential processes a batch of requests sequentially.
// Returns a batch of corresponding responses (or errors).
func Sequential(
	handler  miruken.Handler,
	requests ...any,
) *promise.Promise[[]either.Either[error, any]] {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	return sendBatch(handler, SequentialBatch{requests})
}

// Concurrent processes a batch of requests concurrently.
// Returns a batch of corresponding responses (or errors).
func Concurrent(
	handler  miruken.Handler,
	requests ...any,
) *promise.Promise[[]either.Either[error, any]] {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	return sendBatch(handler, ConcurrentBatch{requests})
}

func sendBatch(
	handler miruken.Handler,
	batch   any,
) *promise.Promise[[]either.Either[error, any]] {
	if r, pr, err := Send[ScheduledResult](handler, batch); err != nil {
		return promise.Reject[[]either.Either[error, any]](err)
	} else if pr != nil {
		return promise.Then(pr, func(result ScheduledResult) []either.Either[error, any] {
			return result.Responses
		})
	} else {
		return promise.Resolve(r.Responses)
	}
}


// Failure returns a new failed result.
func Failure(val error) either.Either[error, any] {
	return either.Left(val)
}

// Success returns a new successful result.
func Success[R any](val R) either.Either[error, R] {
	return either.Right(val)
}
