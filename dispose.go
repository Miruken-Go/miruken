package miruken

type (
	Disposable interface {
		Dispose()
	}
	DisposableFunc func()
)

func (f DisposableFunc) Dispose() {
	f()
}
