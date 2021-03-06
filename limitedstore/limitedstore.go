// Package limitedstore provides a storage with a limited space.
//
// This goal is achieved by using a simple wrapper around existing
// datastores (tusd.DataStore) while limiting the used storage size.
// It will start terminating existing uploads if not enough space is left in
// order to create a new upload.
// The order in which the uploads will be terminated is defined by their size,
// whereas the biggest ones are deleted first.
// This package's functionality is very limited and naive. It will terminate
// uploads whether they are finished yet or not. Only one datastore is allowed to
// access the underlying storage else the limited store will not function
// properly. Two tusd.FileStore instances using the same directory, for example.
// In addition the limited store will keep a list of the uploads' IDs in memory
// which may create a growing memory leak.
//
// While LimitedStore implements the GetReader, LockUpload, UnlockUpload,
// FinishUpload and ConcatUploads methods, it does not contain proper definitions
// for them. When invoked, the call will be passed to the underlying
// data store as long as it provides these methods. If not, either an error
// is returned or nothing happens (see the specific methods for more
// detailed information). The motivation behind this decision was, that this
// allows to expose the additional extensions implemented using the
// interfaces, such as GetReaderDataStore.
package limitedstore

import (
	"github.com/tus/tusd"
	"io"
	"sort"
	"sync"
)

type LimitedStore struct {
	StoreSize int64
	tusd.TerminaterDataStore

	uploads  map[string]int64
	usedSize int64

	mutex *sync.Mutex
}

// pair structure to perform map-sorting
type pair struct {
	key   string
	value int64
}

type pairlist []pair

func (p pairlist) Len() int           { return len(p) }
func (p pairlist) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p pairlist) Less(i, j int) bool { return p[i].value < p[j].value }

// New creates a new limited store with the given size as the maximum storage
// size. The wrapped data store needs to implement the TerminaterDataStore
// interface, in order to provide the required Terminate method.
func New(storeSize int64, dataStore tusd.TerminaterDataStore) *LimitedStore {
	return &LimitedStore{
		StoreSize:           storeSize,
		TerminaterDataStore: dataStore,
		uploads:             make(map[string]int64),
		mutex:               new(sync.Mutex),
	}
}

func (store *LimitedStore) NewUpload(info tusd.FileInfo) (string, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if err := store.ensureSpace(info.Size); err != nil {
		return "", err
	}

	id, err := store.TerminaterDataStore.NewUpload(info)
	if err != nil {
		return "", err
	}

	store.usedSize += info.Size
	store.uploads[id] = info.Size

	return id, nil
}

func (store *LimitedStore) Terminate(id string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	return store.terminate(id)
}

func (store *LimitedStore) terminate(id string) error {
	err := store.TerminaterDataStore.Terminate(id)
	if err != nil {
		return err
	}

	size := store.uploads[id]
	delete(store.uploads, id)
	store.usedSize -= size

	return nil
}

// Ensure enough space is available to store an upload of the specified size.
// It will terminate uploads until enough space is freed.
func (store *LimitedStore) ensureSpace(size int64) error {
	if (store.usedSize + size) <= store.StoreSize {
		// Enough space is available to store the new upload
		return nil
	}

	sortedUploads := make(pairlist, len(store.uploads))
	i := 0
	for u, h := range store.uploads {
		sortedUploads[i] = pair{u, h}
		i++
	}
	sort.Sort(sort.Reverse(sortedUploads))

	// Forward traversal through the uploads in terms of size, biggest upload first
	for _, k := range sortedUploads {
		id := k.key

		if err := store.terminate(id); err != nil {
			return err
		}

		if (store.usedSize + size) <= store.StoreSize {
			// Enough space has been freed to store the new upload
			return nil
		}
	}

	return nil
}

// GetReader will pass the call to the underlying data store if it implements
// the tusd.GetReaderDataStore interface. Else tusd.ErrNotImplemented will be
// returned.
func (store *LimitedStore) GetReader(id string) (io.Reader, error) {
	if s, ok := store.TerminaterDataStore.(tusd.GetReaderDataStore); ok {
		return s.GetReader(id)
	} else {
		return nil, tusd.ErrNotImplemented
	}
}

// LockUpload will pass the call to the underlying data store if it implements
// the tusd.LockerDataStore interface. Else this function simply returns nil.
func (store *LimitedStore) LockUpload(id string) error {
	if s, ok := store.TerminaterDataStore.(tusd.LockerDataStore); ok {
		return s.LockUpload(id)
	}

	return nil
}

// UnlockUpload will pass the call to the underlying data store if it implements
// the tusd.LockerDataStore interface. Else this function simply returns nil.
func (store *LimitedStore) UnlockUpload(id string) error {
	if s, ok := store.TerminaterDataStore.(tusd.LockerDataStore); ok {
		return s.UnlockUpload(id)
	}

	return nil
}

// FinishUpload will pass the call to the underlying data store if it implements
// the tusd.FinisherDataStore interface. Else this function simply returns nil.
func (store *LimitedStore) FinishUpload(id string) error {
	if s, ok := store.TerminaterDataStore.(tusd.FinisherDataStore); ok {
		return s.FinishUpload(id)
	}

	return nil
}

// ConcatUploads will pass the call to the underlying data store if it implements
// the tusd.ConcaterDataStore interface. Else tusd.ErrNotImplemented will be
// returned.
func (store *LimitedStore) ConcatUploads(dest string, src []string) error {
	if s, ok := store.TerminaterDataStore.(tusd.ConcaterDataStore); ok {
		return s.ConcatUploads(dest, src)
	} else {
		return tusd.ErrNotImplemented
	}
}
