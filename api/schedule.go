package api

import (
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/creates"
	"github.com/miruken-go/miruken/either"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
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
		Responses []either.Monad[error, any]
	}

	// Scheduler performs the scheduling of requests.
	Scheduler struct {}
)


// Scheduler

func (s *Scheduler) Constructor(
	_*struct{
		provides.It
		provides.Single
	  },
) {
}

func (s *Scheduler) Concurrent(
	_ *handles.It, concurrent ConcurrentBatch,
	composer miruken.Handler,
) *promise.Promise[ScheduledResult] {
	return promise.New(func(resolve func(ScheduledResult), reject func(error)) {
		requests := concurrent.Requests
		responses := make([]either.Monad[error, any], len(requests))

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

func (s *Scheduler) Sequential(
	_ *handles.It, sequential SequentialBatch,
	composer miruken.Handler,
) *promise.Promise[ScheduledResult] {
	return promise.New(func(resolve func(ScheduledResult), reject func(error)) {
		requests := sequential.Requests
		var responses []either.Monad[error, any]

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

func (s *Scheduler) Publish(
	_ *handles.It, publish Published,
	composer miruken.Handler,
) (p *promise.Promise[any], err error) {
	return Publish(composer, publish.Message)
}

func (s *Scheduler) New(
	_*struct{
		cb creates.It `key:"api.ConcurrentBatch"`
		sb creates.It `key:"api.SequentialBatch"`
		sr creates.It `key:"api.ScheduledResult"`
		p  creates.It `key:"api.Published"`
	  }, create *creates.It,
) any {
	switch create.Key() {
	case "api.ConcurrentBatch":
		return new(ConcurrentBatch)
	case "api.SequentialBatch":
		return new(SequentialBatch)
	case "api.ScheduledResult":
		return new(ScheduledResult)
	case "api.Published":
		return new(Published)
	}
	return nil
}


// Sequential processes a batch of requests sequentially.
// Returns a batch of corresponding responses (or errors).
func Sequential(
	handler  miruken.Handler,
	requests ...any,
) *promise.Promise[[]either.Monad[error, any]] {
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
) *promise.Promise[[]either.Monad[error, any]] {
	if miruken.IsNil(handler) {
		panic("handler cannot be nil")
	}
	return sendBatch(handler, ConcurrentBatch{requests})
}

func process(
	request any,
	handler miruken.Handler,
) (either.Monad[error, any], bool) {
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

func sendBatch(
	handler miruken.Handler,
	batch   any,
) *promise.Promise[[]either.Monad[error, any]] {
	if r, pr, err := Send[ScheduledResult](handler, batch); err != nil {
		return promise.Reject[[]either.Monad[error, any]](err)
	} else if pr != nil {
		return promise.Then(pr, func(result ScheduledResult) []either.Monad[error, any] {
			return result.Responses
		})
	} else {
		return promise.Resolve(r.Responses)
	}
}
