package shutdown

// Disposable is the interface to be used for graceful shutdown
type Disposable interface {
	Dispose()
}
