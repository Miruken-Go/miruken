package test

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/promise"
	"github.com/stretchr/testify/suite"
	"testing"
)

type (
	SendEmail any
	ConfirmSend string
	FailSend    string
	FailConfirm string
	EmailHandler struct {}
	EmailBatcher struct {
		messages []string
		promises []*promise.Promise[any]
		resolves []func()
	}
)

func (e *EmailHandler) Send(
	_*miruken.Handles, send SendEmail,
	composer miruken.Handler,
) any {
	if batch := miruken.GetBatch[*EmailBatcher](composer); batch != nil {
		return batch.Send(nil, send)
	}
	return send
}

func (e *EmailHandler) ConfirmSend(
	_*miruken.Handles, confirm ConfirmSend,
	composer miruken.Handler,
) *promise.Promise[any] {
	if batch := miruken.GetBatch[*EmailBatcher](composer); batch != nil {
		return batch.ConfirmSend(nil, confirm)
	}
	return promise.Resolve[any](string(confirm))
}

func (e *EmailHandler) FailSend(
	_*miruken.Handles, fail FailSend,
) any {
	panic("can't send message")
}

func (e *EmailHandler) FailConfirm(
	_*miruken.Handles, fail FailConfirm,
	composer miruken.Handler,
) any {
	if batch := miruken.GetBatch[*EmailBatcher](composer); batch != nil {
		return batch.FailConfirm(nil, fail)
	}
	return promise.Resolve[any](nil)
}

func (e *EmailBatcher) Send(
	_*miruken.Handles, send SendEmail,
) any {
	message := fmt.Sprintf("%v batch", send)
	e.messages = append(e.messages, message)
	return nil
}

func (e *EmailBatcher) ConfirmSend(
	_*miruken.Handles, confirm ConfirmSend,
) *promise.Promise[any]  {
	e.messages = append(e.messages, string(confirm))
	p := promise.New(func(resolve func(any), reject func(error)) {
		e.resolves = append(e.resolves, func() {
			resolve(fmt.Sprintf("%v batch", confirm))
		})
	})
	e.promises = append(e.promises, p)
	return p
}

func (e *EmailBatcher) FailConfirm(
	_*miruken.Handles, fail FailConfirm,
) *promise.Promise[any] {
	p := promise.New(func(resolve func(any), reject func(error)) {
		e.resolves = append(e.resolves, func() {
			reject(errors.New("can't send message"))
		})
	})
	e.promises = append(e.promises, p)
	return p
}

func (e *EmailBatcher) CompleteBatch(
	composer miruken.Handler,
) (any, *promise.Promise[any], error) {
	if e.messages == nil {
		return nil, nil, nil
	}
	for _, resolve := range  e.resolves {
		resolve()
	}
	if r, pr, err := api.Send[any](composer, SendEmail(e.messages)); err != nil {
		return nil, nil, err
	} else {
		if promises := e.promises; len(promises) > 0 {
			return nil, promise.Then(promise.All(promises...), func([]any) any {
				if pr != nil {
					if r, err = pr.Await(); err != nil {
						panic(err)
					}
				}
				return r
			}), nil
		}
		return r,pr, nil
	}
}

type BatchTestSuite struct {
	suite.Suite
}

func (suite *BatchTestSuite) Setup() (miruken.Handler, error) {
	return miruken.Setup(
		miruken.HandlerSpecs(&EmailHandler{}),
		api.Feature())
}

func (suite *BatchTestSuite) TestBatch() {
	suite.Run("Uses Same Batcher", func() {
		handler, _ := miruken.Setup(miruken.Handlers(new(EmailHandler)))
		miruken.Batch(handler, func(batch miruken.Handler) {
			b := miruken.GetBatch[*EmailBatcher](batch)
			suite.NotNil(b)
			suite.Same(b, miruken.GetBatch[*EmailBatcher](batch))
		})
	})

	suite.Run("Batch", func() {
		handler, _ := suite.Setup()
		r, pr, err := api.Send[string](handler, SendEmail("hello"))
		suite.Nil(err)
		suite.Nil(pr)
		suite.Equal("hello", r)
		results, err := miruken.Batch(handler, func(batch miruken.Handler) {
			r, pr, err := api.Send[string](batch, SendEmail("hello"))
			suite.Nil(err)
			suite.Zero(r)
			suite.Nil(pr)
		}).Await()
		suite.Nil(err)
		suite.Len(results, 1)
		suite.Equal("hello batch", results[0])
	})

	suite.Run("Batch Async", func() {
		handler, _ := suite.Setup()
		r, pr, err := api.Send[string](handler, ConfirmSend("hello"))
		suite.Nil(err)
		suite.Zero(r)
		suite.NotNil(pr)
		r, err = pr.Await()
		suite.Nil(err)
		suite.Equal("hello", r)
		count := 0
		results, err := miruken.Batch(handler, func(batch miruken.Handler) {
			r, pr, err := api.Send[string](batch, ConfirmSend("hello"))
			suite.Nil(err)
			suite.Zero(r)
			suite.NotNil(pr)
			promise.Then(pr, func(res string) any {
				suite.Equal("hello batch", res)
				count++
				return nil
			})
		}).Await()
		suite.Nil(err)
		suite.Len(results, 1)
		suite.Equal("hello", results[0])
	})
}

func TestBatchTestSuite(t *testing.T) {
	suite.Run(t, new(BatchTestSuite))
}
