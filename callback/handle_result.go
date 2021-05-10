package callback

var (
	Handled           = HandleResult{true, false}
	HandledAndStop    = HandleResult{true, true}
	NotHandled        = HandleResult{false, false}
	NotHandledAndStop = HandleResult{false, true}
)

type HandleResult struct {
	Handled bool
	Stop    bool
}

type HandleResultBlock func(HandleResult) (HandleResult, error)

func (r HandleResult) Then(
	block HandleResultBlock,
) (HandleResult, error) {
	if block == nil {
		panic("nil block")
	}

	if r.Stop {
		return r, nil
	} else {
		if b, err := block(r); err != nil {
			return r, err
		} else {
			return r.Or(b), nil
		}
	}
}

func (r HandleResult) ThenIf(
	condition bool,
	block     HandleResultBlock,
) (HandleResult, error) {
	if block == nil {
		panic("nil block")
	}

	if r.Stop || !condition {
		return r, nil
	} else {
		if b, err := block(r); err != nil {
			return r, err
		} else {
			return r.Or(b), nil
		}
	}
}

func (r HandleResult) Otherwise(
	block HandleResultBlock,
) (HandleResult, error) {
	if block == nil {
		panic("nil block")
	}

	if r.Handled || r.Stop {
		return r, nil
	} else {
		return block(r)
	}
}

func (r HandleResult) OtherwiseIf(
	condition bool,
	block     HandleResultBlock,
) (HandleResult, error) {
	if block == nil {
		panic("nil block")
	}

	if (r.Handled || r.Stop) && !condition {
		return r, nil
	} else {
		if b, err := block(r); err != nil {
			return r, err
		} else {
			return r.Or(b), nil
		}
	}
}

func (r HandleResult) Or(other HandleResult) HandleResult {
	if r.Handled || other.Handled {
		if r.Stop || other.Stop {
			return HandledAndStop
		} else {
			return Handled
		}
	} else {
		if r.Stop || other.Stop {
			return NotHandledAndStop
		} else {
			return NotHandled
		}
	}
}

func (r HandleResult) And(other HandleResult) HandleResult {
	if r.Handled && other.Handled {
		if r.Stop || other.Stop {
			return HandledAndStop
		} else {
			return Handled
		}
	} else {
		if r.Stop || other.Stop {
			return NotHandledAndStop
		} else {
			return NotHandled
		}
	}
}

