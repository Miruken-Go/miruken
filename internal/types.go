package internal

var (
	AnyType      = TypeOf[any]()
	AnySliceType = TypeOf[[]any]()
	ErrorType    = TypeOf[error]()
)

