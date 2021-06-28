package carstore

import (
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car/v2/blockstore"
	"golang.org/x/xerrors"
)

// CarReadWriteStoreTracker tracks the lifecycle of a ReadWrite CAR Blockstore and makes it easy to create/get/cleanup the blockstores.
// It's important to close a CAR Blockstore when done using it so that the backing CAR file can be closed.
type CarReadWriteStoreTracker struct {
	mu     sync.RWMutex
	stores map[string]*blockstore.ReadWrite
}

func NewCarReadWriteStoreTracker() *CarReadWriteStoreTracker {
	return &CarReadWriteStoreTracker{
		stores: make(map[string]*blockstore.ReadWrite),
	}
}

func (r *CarReadWriteStoreTracker) GetOrCreate(key string, carV2FilePath string, rootCid cid.Cid) (*blockstore.ReadWrite, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if bs, ok := r.stores[key]; ok {
		return bs, nil
	}

	rwBs, err := blockstore.NewReadWrite(carV2FilePath, []cid.Cid{rootCid})
	if err != nil {
		return nil, xerrors.Errorf("failed to create read-write blockstore: %w", err)
	}
	r.stores[key] = rwBs

	return rwBs, nil
}

func (r *CarReadWriteStoreTracker) Get(key string) (*blockstore.ReadWrite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if bs, ok := r.stores[key]; ok {
		return bs, nil
	}

	return nil, xerrors.Errorf("could not get blockstore for key %s: %w", ErrNotFound)
}

func (r *CarReadWriteStoreTracker) CleanBlockstore(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if bs, ok := r.stores[key]; ok {
		delete(r.stores, key)

		// calling a Finalize on a read-write blockstore is equivalent to closing it.
		if err := bs.Finalize(); err != nil {
			return xerrors.Errorf("finalize call failed: %w", err)
		}
	}

	return nil
}
