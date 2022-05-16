package undead

import "sync"

// type to implement a lock
type Mutex struct {
	mu sync.Mutex
}

// Lock mutex m in routine r
// TODO: change so that r is calculated and taken from Routines
func (m *Mutex) Lock(r *Routine) {
	defer m.mu.Lock()

	// if detection is disabled
	if !Opts.RunDetection {
		return
	}

	// update data structures
	r.updateLock(m)

}

// Unlock mutex m
func (m *Mutex) Unlock() {
	defer m.mu.Unlock()

}
