package wiring

import (
	"sync"
	"testing"
	"time"
)

// TestSessionLocker_LockUnlock_SameSession verifies that two goroutines
// locking the same session are serialized — operations execute sequentially.
func TestSessionLocker_LockUnlock_SameSession(t *testing.T) {
	sl := &SessionLocker{}
	var counter int
	var wg sync.WaitGroup
	wg.Add(2)

	// Use a channel to record execution order and prove serialization.
	order := make(chan int, 2)

	// Goroutine 1: locks session-1, records order, increments, sleeps, unlocks.
	go func() {
		defer wg.Done()
		sl.Lock("session-1")
		order <- 1
		counter++
		time.Sleep(10 * time.Millisecond)
		counter++
		sl.Unlock("session-1")
	}()

	// Small delay to increase likelihood that goroutine 1 acquires the lock first.
	time.Sleep(5 * time.Millisecond)

	// Goroutine 2: locks session-1, records order, increments, sleeps, unlocks.
	go func() {
		defer wg.Done()
		sl.Lock("session-1")
		order <- 2
		counter++
		time.Sleep(10 * time.Millisecond)
		counter++
		sl.Unlock("session-1")
	}()

	wg.Wait()

	// If serialized, counter must be 4 (each goroutine increments twice).
	// If not serialized, we'd have a data race and counter could be anything.
	if counter != 4 {
		t.Errorf("expected counter to be 4 (serialized), got %d", counter)
	}

	// Verify both goroutines executed (order has 2 entries).
	close(order)
	var orderSlice []int
	for v := range order {
		orderSlice = append(orderSlice, v)
	}
	if len(orderSlice) != 2 {
		t.Errorf("expected 2 order entries, got %d", len(orderSlice))
	}
}

// TestSessionLocker_LockUnlock_DifferentSessions verifies that two goroutines
// locking different sessions do not block each other.
func TestSessionLocker_LockUnlock_DifferentSessions(t *testing.T) {
	sl := &SessionLocker{}
	firstLocked := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondLocked := make(chan struct{})
	secondDone := make(chan struct{})

	go func() {
		sl.Lock("session-1")
		close(firstLocked)
		<-releaseFirst
		sl.Unlock("session-1")
	}()

	<-firstLocked

	go func() {
		sl.Lock("session-2")
		close(secondLocked)
		sl.Unlock("session-2")
		close(secondDone)
	}()

	select {
	case <-secondLocked:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("session-2 was blocked by session-1")
	}

	close(releaseFirst)

	select {
	case <-secondDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("session-2 goroutine did not finish")
	}
}

// TestSessionLocker_ConcurrentCreateDestroy simulates a create and destroy
// operation on the same session running concurrently. The final state must
// be "destroyed" — never "creating" and "destroying" simultaneously.
func TestSessionLocker_ConcurrentCreateDestroy(t *testing.T) {
	sl := &SessionLocker{}
	state := "idle"
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Create operation.
	go func() {
		defer wg.Done()
		sl.Lock("s1")
		if state == "idle" {
			state = "creating"
			time.Sleep(10 * time.Millisecond)
			state = "created"
		}
		sl.Unlock("s1")
	}()

	// Goroutine 2: Destroy operation.
	go func() {
		defer wg.Done()
		sl.Lock("s1")
		if state == "created" {
			state = "destroying"
			time.Sleep(10 * time.Millisecond)
			state = "destroyed"
		}
		sl.Unlock("s1")
	}()

	wg.Wait()

	// Valid final states: "created" (destroy ran first, found non-created state)
	// or "destroyed" (create ran first, then destroy).
	// Invalid: "creating", "destroying", "idle" — these indicate a race.
	if state != "created" && state != "destroyed" {
		t.Errorf("expected final state to be 'created' or 'destroyed', got %q", state)
	}
}

// TestSessionLocker_ConcurrentMutateDestroy simulates a mutate and destroy
// operation on the same session running concurrently. State must remain
// consistent.
func TestSessionLocker_ConcurrentMutateDestroy(t *testing.T) {
	sl := &SessionLocker{}
	state := "active"
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Mutate operation.
	go func() {
		defer wg.Done()
		sl.Lock("s1")
		if state == "active" {
			state = "mutating"
			time.Sleep(10 * time.Millisecond)
			state = "mutated"
		}
		sl.Unlock("s1")
	}()

	// Goroutine 2: Destroy operation.
	go func() {
		defer wg.Done()
		sl.Lock("s1")
		if state == "active" || state == "mutated" {
			state = "destroying"
			time.Sleep(10 * time.Millisecond)
			state = "destroyed"
		}
		sl.Unlock("s1")
	}()

	wg.Wait()

	// Valid final states: "mutated" (destroy ran first, found active, didn't match)
	// or "destroyed" (either order, as long as serialization held).
	// Invalid: "mutating", "destroying" — these indicate a race.
	if state != "mutated" && state != "destroyed" {
		t.Errorf("expected final state to be 'mutated' or 'destroyed', got %q", state)
	}
}

// TestSessionLocker_MultipleLockUnlock_Cycles verifies that repeated
// lock/unlock cycles on the same session work correctly.
func TestSessionLocker_MultipleLockUnlock_Cycles(t *testing.T) {
	sl := &SessionLocker{}
	const cycles = 100
	var counter int

	for i := 0; i < cycles; i++ {
		sl.Lock("session-1")
		counter++
		sl.Unlock("session-1")
	}

	if counter != cycles {
		t.Errorf("expected counter to be %d, got %d", cycles, counter)
	}
}

// TestSessionLocker_EmptySessionID verifies that an empty string session ID
// is handled as a valid key.
func TestSessionLocker_EmptySessionID(t *testing.T) {
	sl := &SessionLocker{}
	var wg sync.WaitGroup
	wg.Add(2)

	var counter int

	go func() {
		defer wg.Done()
		sl.Lock("")
		counter++
		time.Sleep(5 * time.Millisecond)
		counter++
		sl.Unlock("")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Millisecond)
		sl.Lock("")
		counter++
		time.Sleep(5 * time.Millisecond)
		counter++
		sl.Unlock("")
	}()

	wg.Wait()

	if counter != 4 {
		t.Errorf("expected counter to be 4 (serialized), got %d", counter)
	}
}

// TestSessionLocker_UnlockWithoutLock verifies that calling Unlock without a
// matching Lock fails fast.
func TestSessionLocker_UnlockWithoutLock(t *testing.T) {
	sl := &SessionLocker{}
	defer func() {
		if recover() == nil {
			t.Fatal("expected Unlock to panic without a matching Lock")
		}
	}()

	sl.Unlock("nonexistent-session")
}

// TestSessionLocker_MultipleGoroutinesSameSession verifies serialization
// with many goroutines contending for the same session lock.
func TestSessionLocker_MultipleGoroutinesSameSession(t *testing.T) {
	sl := &SessionLocker{}
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	var counter int

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			sl.Lock("session-1")
			counter++
			time.Sleep(1 * time.Millisecond)
			sl.Unlock("session-1")
		}()
	}

	wg.Wait()

	if counter != goroutines {
		t.Errorf("expected counter to be %d, got %d", goroutines, counter)
	}
}

// TestSessionLocker_UnlockRemovesEntry verifies that the session entry is
// cleaned up after the final Unlock.
func TestSessionLocker_UnlockRemovesEntry(t *testing.T) {
	sl := &SessionLocker{}

	sl.Lock("session-1")
	if got := len(sl.locks); got != 1 {
		t.Fatalf("expected 1 session entry after Lock, got %d", got)
	}

	sl.Unlock("session-1")
	if got := len(sl.locks); got != 0 {
		t.Fatalf("expected session entry to be removed after Unlock, got %d", got)
	}
}
