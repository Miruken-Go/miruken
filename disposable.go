package miruken

type Disposable interface {
	Dispose()
}

type DisposableFunc func()

func (f DisposableFunc) Dispose() {
	f()
}
