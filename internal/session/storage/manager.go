package storage

import "sync"

// Unsafe stores all the session state.
// By itself it does not provide any concurrency guarantees: if you need them, use a `Manager` instead.
type Unsafe struct {
	sessions    map[SessionId]*session
	inviteCodes map[InviteCode]SessionId
}

// A Manager encapsulates an `Unsafe` and provides a thread-safe interface to the storage.
type Manager struct {
	mtx   sync.Mutex
	inner Unsafe
}

// Atomically performs the provided operation on atomically.
// While the function is being run, no other goroutine may access the inner storage.
// This function is not re-entrant: do not call `Atomically` in `f`.
func Atomically[R any](m *Manager, f func(s *Unsafe) R) R {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	return f(&m.inner)
}

func (s *Unsafe) PlayerCount(sid SessionId) int {
	return len(s.sessions[sid].players)
}
