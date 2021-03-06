// Package memorylocker provides an in-memory locking mechanism.
//
// When multiple processes are attempting to access an upload, whether it be
// by reading or writing, a syncronization mechanism is required to prevent
// data corruption, especially to ensure correct offset values and the proper
// order of chunks inside a single upload.
//
// MemoryLocker persists locks using memory and therefore allowing a simple and
// cheap mechansim. Locks will only exist as long as this object is kept in
// reference and will be erased if the program exits.
package memorylocker

import (
	"github.com/tus/tusd"
)

// MemoryLocker persists locks using memory and therefore allowing a simple and
// cheap mechansim. Locks will only exist as long as this object is kept in
// reference and will be erased if the program exits.
type MemoryLocker struct {
	tusd.DataStore
	locks map[string]bool
}

// New creates a new lock memory wrapper around the provided storage.
func NewMemoryLocker(store tusd.DataStore) *MemoryLocker {
	return &MemoryLocker{
		DataStore: store,
		locks:     make(map[string]bool),
	}
}

// LockUpload tries to obtain the exclusive lock.
func (locker *MemoryLocker) LockUpload(id string) error {

	// Ensure file is not locked
	if _, ok := locker.locks[id]; ok {
		return tusd.ErrFileLocked
	}

	locker.locks[id] = true

	return nil
}

// UnlockUpload releases a lock. If no such lock exists, no error will be returned.
func (locker *MemoryLocker) UnlockUpload(id string) error {
	// Deleting a non-existing key does not end in unexpected errors or panic
	// since this operation results in a no-op
	delete(locker.locks, id)

	return nil
}
