package test

import (
	"errors"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/setup"
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
	_ *handles.It, send SendEmail,
	composer miruken.Handler,
) any {
	if batch := miruken.GetBatch[*EmailBatcher](composer); batch != nil {
		return batch.Send(nil, send)
	}
	return send
}

func (e *EmailHandler) ConfirmSend(
	_ *handles.It, confirm ConfirmSend,
	composer miruken.Handler,
) *promise.Promise[any] {
	if batch := miruken.GetBatch[*EmailBatcher](composer); batch != nil {
		return batch.ConfirmSend(nil, confirm)
	}
	return promise.Resolve[any](string(confirm))
}

func (e *EmailHandler) FailSend(
	_ *handles.It, fail FailSend,
) any {
	panic("can't send message")
}

func (e *EmailHandler) FailConfirm(
	_ *handles.It, fail FailConfirm,
	composer miruken.Handler,
) any {
	if batch := miruken.GetBatch[*EmailBatcher](composer); batch != nil {
		return batch.FailConfirm(nil, fail)
	}
	return promise.Resolve[any](nil)
}

func (e *EmailBatcher) Send(
	_ *handles.It, send SendEmail,
) any {
	message := fmt.Sprintf("%v batch", send)
	e.messages = append(e.messages, message)
	return nil
}

func (e *EmailBatcher) ConfirmSend(
	_ *handles.It, confirm ConfirmSend,
) *promise.Promise[any]  {
	e.messages = append(e.messages, string(confirm))
	d := promise.Defer[any]()
	e.resolves = append(e.resolves, func() {
		d.Resolve(fmt.Sprintf("%v batch", confirm))
	})
	p := d.Promise()
	e.promises = append(e.promises, p)
	return p
}

func (e *EmailBatcher) FailConfirm(
	_*handles.It, fail FailConfirm,
) *promise.Promise[any] {
	d := promise.Defer[any]()
	e.resolves = append(e.resolves, func() {
		d.Reject(errors.New("can't send message"))
	})
	p := d.Promise()
	e.promises = append(e.promises, p)
	return p
}

func (e *EmailBatcher) CompleteBatch(
	composer miruken.Handler,
) (any, *promise.Promise[any], error) {
	for _, resolve := range e.resolves {
		resolve()
	}
	messages := e.messages
	if messages == nil {
		messages = []string{}
	}
	if r, pr, err := api.Send[any](composer, SendEmail(messages)); err != nil {
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
		return r, pr, nil
	}
}

type BatchTestSuite struct {
	suite.Suite
}

func (suite *BatchTestSuite) Setup() (miruken.Handler, error) {
	return setup.New(api.Feature()).
		Specs(&EmailHandler{}).
		Handler()
}

func (suite *BatchTestSuite) TestBatch() {
	suite.Run("Uses Same Batcher", func() {
		handler, _ := setup.New().Handlers(new(EmailHandler)).Handler()
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
		suite.Equal([]string{"hello batch"}, results[0])
	})

	suite.Run("Batch Async", func() {
		handler, _ := suite.Setup()
		r, pr, err := api.Send[string](handler, ConfirmSend("hello"))
		suite.Nil(err)
		suite.NotNil(pr)
		r, err = pr.Await()
		suite.Nil(err)
		suite.Equal("hello", r)
		results, err := miruken.Batch(handler, func(batch miruken.Handler) {
			_, pr, err := api.Send[string](batch, ConfirmSend("hello"))
			suite.Nil(err)
			suite.NotNil(pr)
			promise.Then(pr, func(res string) any {
				suite.Equal("hello batch", res)
				return nil
			})
		}).Await()
		suite.Nil(err)
		suite.Len(results, 1)
		suite.Equal([]string{"hello"}, results[0])
	})

	suite.Run("Batch Fail Async", func() {
		handler, _ := suite.Setup()
		count := 0
		results, err := miruken.Batch(handler, func(batch miruken.Handler) {
			_, pr, err := api.Send[string](batch, FailConfirm("hello"))
			suite.Nil(err)
			suite.NotNil(pr)
			promise.Catch(pr, func(err error) error {
				suite.Equal("can't send message", err.Error())
				count++
				return nil
			})
		}).Await()
		suite.NotNil(err)
		suite.Nil(results)
	})

	suite.Run("No Batch After Await", func() {
		handler, _ := suite.Setup()
		results, err := miruken.Batch(handler, func(batch miruken.Handler) {
			_, pr, err := api.Send[string](batch, ConfirmSend("hello"))
			suite.Nil(err)
			suite.NotNil(pr)
			promise.Then(pr, func(res string) any {
				suite.Equal("hello batch", res)
				return nil
			})
		}).Await()
		suite.Nil(err)
		suite.Len(results, 1)
		suite.Equal([]string{"hello"}, results[0])

		r, pr, err := api.Send[string](handler, ConfirmSend("hello"))
		suite.Nil(err)
		suite.NotNil(pr)
		r, err = pr.Await()
		suite.Nil(err)
		suite.Equal("hello", r)
	})

	suite.Run("No Batch After Completed", func() {
		count := 0
		handler, _ := suite.Setup()
		results, err := miruken.BatchAsync(handler, func(batch miruken.Handler) *promise.Promise[any] {
			_, pr, err := api.Send[string](batch, ConfirmSend("hello"))
			suite.Nil(err)
			suite.NotNil(pr)
			return promise.Then(pr, func(res string) any {
				r1, pr1, err1 := api.Send[string](batch, ConfirmSend("hello"))
				suite.Nil(err)
				suite.NotNil(pr)
				r1, err1 = pr1.Await()
				suite.Nil(err1)
				suite.Equal("hello", r1)
				count = 1
				return nil
			})
		}).Await()
		suite.Nil(err)
		suite.Len(results, 1)
		suite.Equal(1, count)
	})
}

func TestBatchTestSuite(t *testing.T) {
	suite.Run(t, new(BatchTestSuite))
}
