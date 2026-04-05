package wiring

import (
	"fmt"
	"sync"
)

// SessionLocker serializes operations on the same session while allowing
// independent sessions to proceed in parallel.
//
// It is zero-value ready. Entries are removed after the last matching Unlock.
// Unlock panics if called without a matching Lock.
type SessionLocker struct {
	mu    sync.Mutex
	locks map[string]*sessionLock
}

type sessionLock struct {
	mu   sync.Mutex
	refs int
}

// Lock acquires the mutex for the given session. If the session's mutex does
// not yet exist, it is created. Callers must call Unlock(sessionID) when done.
//
// The recommended pattern is:
//
//	sl.Lock(sessionID)
//	defer sl.Unlock(sessionID)
func (sl *SessionLocker) Lock(sessionID string) {
	sl.mu.Lock()
	if sl.locks == nil {
		sl.locks = make(map[string]*sessionLock)
	}
	entry := sl.locks[sessionID]
	if entry == nil {
		entry = &sessionLock{}
		sl.locks[sessionID] = entry
	}
	entry.refs++
	sl.mu.Unlock()

	entry.mu.Lock()
}

// Unlock releases the mutex for the given session.
func (sl *SessionLocker) Unlock(sessionID string) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	entry, ok := sl.locks[sessionID]
	if !ok {
		panic(fmt.Sprintf("unlock of unknown session %q", sessionID))
	}

	entry.mu.Unlock()
	entry.refs--
	if entry.refs == 0 {
		delete(sl.locks, sessionID)
	}
}
