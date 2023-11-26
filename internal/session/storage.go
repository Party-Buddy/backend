package session

import "sync"

// A SyncStorage encapsulates an [UnsafeStorage] and provides a thread-safe interface to the storage.
type SyncStorage struct {
	mtx   sync.Mutex
	inner UnsafeStorage
}

func NewSyncStorage() SyncStorage {
	return SyncStorage{
		mtx:   sync.Mutex{},
		inner: NewUnsafeStorage(),
	}
}

// Atomically performs the provided operation on the inner storage atomically.
// While the function is being run, no other goroutine may access the inner storage.
// This function is not re-entrant: do not call Atomically in `f`.
func (s *SyncStorage) Atomically(f func(s *UnsafeStorage)) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	f(&s.inner)
}
