package wiring

import "sync"

// SessionLocker serializes operations on the same session while allowing
// independent sessions to proceed in parallel.
//
// It manages a registry of per-session sync.Mutex instances using sync.Map
// internally. No package-level mutable state is used — the registry is a
// field on the struct, instantiated once and passed to consumers.
type SessionLocker struct {
	mu sync.Map // map[string]*sync.Mutex
}

// Lock acquires the mutex for the given session. If the session's mutex
// does not yet exist, it is created. Callers must call Unlock(sessionID)
// when done.
//
// The recommended pattern is:
//
//	sl.Lock(sessionID)
//	defer sl.Unlock(sessionID)
func (sl *SessionLocker) Lock(sessionID string) {
	actual, _ := sl.mu.LoadOrStore(sessionID, &sync.Mutex{})
	mu := actual.(*sync.Mutex)
	mu.Lock()
}

// Unlock releases the mutex for the given session.
func (sl *SessionLocker) Unlock(sessionID string) {
	val, ok := sl.mu.Load(sessionID)
	if !ok {
		// Defensive: should not happen if Lock/Unlock are paired correctly.
		return
	}
	mu := val.(*sync.Mutex)
	mu.Unlock()
}
