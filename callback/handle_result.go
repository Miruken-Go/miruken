package callback

import "github.com/hashicorp/go-multierror"

var (
	Handled           = HandleResult{true,  false, nil}
	HandledAndStop    = HandleResult{true,  true,  nil}
	NotHandled        = HandleResult{false, false, nil}
	NotHandledAndStop = HandleResult{false, true,  nil}
)

type HandleResult struct {
	handled bool
	stop    bool
	err     error
}

type HandleResultBlock func(HandleResult) HandleResult

func (r HandleResult) IsHandled() bool {
	return r.handled
}

func (r HandleResult) ShouldStop() bool {
	return r.stop
}

func (r HandleResult) IsError() bool {
	return r.err != nil
}

func (r HandleResult) Error() error {
	return r.err
}

func (r HandleResult) Then(
	block HandleResultBlock,
) HandleResult {
	if block == nil {
		panic("nil block")
	}

	if r.stop {
		return r
	} else {
		return r.Or(block(r))
	}
}

func (r HandleResult) ThenIf(
	condition bool,
	block     HandleResultBlock,
) HandleResult {
	if block == nil {
		panic("nil block")
	}

	if r.stop || !condition {
		return r
	} else {
		return r.Or(block(r))
	}
}

func (r HandleResult) Otherwise(
	block HandleResultBlock,
) HandleResult {
	if block == nil {
		panic("nil block")
	}

	if r.handled || r.stop {
		return r
	} else {
		return block(r)
	}
}

func (r HandleResult) OtherwiseIf(
	condition bool,
	block     HandleResultBlock,
) HandleResult {
	if block == nil {
		panic("nil block")
	}

	if (r.handled || r.stop) && !condition {
		return r
	} else {
		return r.Or(block(r))
	}
}

func (r HandleResult) Or(other HandleResult) HandleResult {
	err := combineErrors(r, other)
	if r.handled || other.handled {
		if r.stop || other.stop {
			return withError(HandledAndStop, err)
		} else {
			return withError(Handled, err)
		}
	} else {
		if r.stop || other.stop {
			return withError(NotHandledAndStop, err)
		} else {
			return withError(NotHandled, err)
		}
	}
}

func (r HandleResult) And(other HandleResult) HandleResult {
	err := combineErrors(r, other)
	if r.handled && other.handled {
		if r.stop || other.stop {
			return withError(HandledAndStop, err)
		} else {
			return withError(Handled, err)
		}
	} else {
		if r.stop || other.stop {
			return withError(NotHandledAndStop, err)
		} else {
			return withError(NotHandled, err)
		}
	}
}

func withError(result HandleResult, err error) HandleResult {
	if err == nil {
		return result
	}
	return HandleResult{result.handled, true, err}
}

func combineErrors(r1 HandleResult, r2 HandleResult) error {
	if e1, e2 := r1.err, r2.err; e1 != nil && e2 != nil {
		return multierror.Append(e1, e2)
	} else if e1 != nil {
		return e1
	} else if e2 != nil {
		return e2
	}
	return nil
}