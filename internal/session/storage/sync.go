package storage

import "sync"

// A SyncStorage encapsulates an [UnsafeStorage] and provides a thread-safe interface to the storage.
type SyncStorage struct {
	mtx   sync.Mutex
	inner UnsafeStorage
}

// Atomically performs the provided operation on the inner storage atomically.
// While the function is being run, no other goroutine may access the inner storage.
// This function is not re-entrant: do not call Atomically in `f`.
func Atomically[R any](s *SyncStorage, f func(s *UnsafeStorage) R) R {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return f(&s.inner)
}
