package deadlock

/*
Copyright (C) 2022  Erik Kassubek

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/*
Author: Erik Kassubek <erik-kassubek@t-online.de>
Package: deadlock
Project: Bachelor Project at the Albert-Ludwigs-University Freiburg,
	Institute of Computer Science: Dynamic Deadlock Detection in Go
*/

/*
mutex.go
This file implements the drop-in-replacement for the locks (mutexes) as well as
the lock and unlock operations for these locks.
*/

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// type to implement a lock
type Mutex struct {
	mu                   sync.Mutex
	context              []callerInfo // info about the creation and lock/unlock of this lock
	in                   bool         // set to true after lock was initialized
	isLocked             bool         // set to true if lock is locked
	isLockedRoutineIndex int          // index of the routine, which holds the lock
	memoryPosition       uintptr      // position of the mutex in memory
}

// create Lock
func NewLock() *Mutex {
	// initialize detector if necessary
	if !initialized {
		initialize()
	}

	m := Mutex{
		in:                   true,
		isLockedRoutineIndex: -1,
	}
	_, file, line, _ := runtime.Caller(1)
	m.context = append(m.context, newInfo(file, line, true, ""))
	m.memoryPosition = uintptr(unsafe.Pointer(&m))

	return &m
}

// ====== GETTER ===============================================================

// getter for isLocked
func (m *Mutex) getIsLocked() *bool {
	return &m.isLocked
}

// getter for isLockedRoutineIndex
func (m *Mutex) getIsLockedRoutineIndex() *int {
	return &m.isLockedRoutineIndex
}

// getter for context
func (m *Mutex) getContext() *[]callerInfo {
	return &m.context
}

// getter for memoryPosition
func (m *Mutex) getMemoryPosition() uintptr {
	return m.memoryPosition
}

// ====== FUNCTIONS ============================================================

// Lock mutex m
func (m *Mutex) Lock() {
	if !m.in {
		errorMessage := fmt.Sprint("Lock ", &m, " was not created. Use ",
			"x := NewLock()")
		panic(errorMessage)
	}

	// initialize detector if necessary
	if !initialized {
		initialize()
	}

	defer func() {
		m.mu.Lock()
		m.isLocked = true
	}()

	// if detection is disabled
	if !opts.periodicDetection && !opts.comprehensiveDetection {
		return
	}

	index := getRoutineIndex()
	if index == -1 {
		// create new routine, if not initialized
		newRoutine()
	}
	index = getRoutineIndex()

	r := &routines[index]

	// check for double locking
	if opts.checkDoubleLocking && m.isLocked {
		r.checkDoubleLocking(m, index)
	}

	m.isLockedRoutineIndex = index

	numRoutine := runtime.NumGoroutine()
	// update data structures if more than on routine is running
	if numRoutine > 1 {
		(*r).updateLock(m, uintptr(unsafe.Pointer(m)))
	}

}

// Trylock mutex m
func (m *Mutex) TryLock() bool {
	if !m.in {
		errorMessage := fmt.Sprint("Lock ", &m, " was not created. Use ",
			"x := NewLock()")
		panic(errorMessage)
	}

	// initialize detector if necessary
	if !initialized {
		initialize()
	}

	res := m.mu.TryLock()

	if res {
		m.isLocked = true
	}

	if !opts.periodicDetection && !opts.comprehensiveDetection {
		return res
	}

	index := getRoutineIndex()
	if index == -1 {
		// create new routine, if not initialized
		newRoutine()
	}

	r := &routines[index]
	if res && opts.checkDoubleLocking {
		r.checkDoubleLocking(m, index)
	}

	m.isLockedRoutineIndex = index

	// update data structures if more than on routine is running
	if runtime.NumGoroutine() > 1 {
		if res {
			(*r).updateTryLock(m)
		}
	}

	return res
}

// Unlock mutex m
func (m *Mutex) Unlock() {
	if !m.isLocked {
		errorMessage := fmt.Sprint("Tried to unLock lock", &m,
			" which was not locked.")
		panic(errorMessage)
	}
	defer func() {
		m.mu.Unlock()
		m.isLockedRoutineIndex = -1
		m.isLocked = false
	}()

	if !opts.periodicDetection && !opts.comprehensiveDetection {
		return
	}

	index := getRoutineIndex()

	r := &routines[index]
	(*r).updateUnlock(m)
}
