package mem

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"sync"

	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/restic"

	"github.com/restic/restic/internal/debug"
)

type memMap map[restic.Handle][]byte

// make sure that MemoryBackend implements backend.Backend
var _ restic.Backend = &MemoryBackend{}

var errNotFound = errors.New("not found")

// MemoryBackend is a mock backend that uses a map for storing all data in
// memory. This should only be used for tests.
type MemoryBackend struct {
	data memMap
	m    sync.Mutex
}

// New returns a new backend that saves all data in a map in memory.
func New() *MemoryBackend {
	be := &MemoryBackend{
		data: make(memMap),
	}

	debug.Log("created new memory backend")

	return be
}

// Test returns whether a file exists.
func (be *MemoryBackend) Test(ctx context.Context, h restic.Handle) (bool, error) {
	be.m.Lock()
	defer be.m.Unlock()

	debug.Log("Test %v", h)

	if _, ok := be.data[h]; ok {
		return true, nil
	}

	return false, nil
}

// IsNotExist returns true if the file does not exist.
func (be *MemoryBackend) IsNotExist(err error) bool {
	return errors.Cause(err) == errNotFound
}

// Save adds new Data to the backend.
func (be *MemoryBackend) Save(ctx context.Context, h restic.Handle, rd io.Reader) error {
	if err := h.Valid(); err != nil {
		return err
	}

	be.m.Lock()
	defer be.m.Unlock()

	if h.Type == restic.ConfigFile {
		h.Name = ""
	}

	if _, ok := be.data[h]; ok {
		return errors.New("file already exists")
	}

	buf, err := ioutil.ReadAll(rd)
	if err != nil {
		return err
	}

	be.data[h] = buf
	debug.Log("saved %v bytes at %v", len(buf), h)

	return nil
}

// Load returns a reader that yields the contents of the file at h at the
// given offset. If length is nonzero, only a portion of the file is
// returned. rd must be closed after use.
func (be *MemoryBackend) Load(ctx context.Context, h restic.Handle, length int, offset int64) (io.ReadCloser, error) {
	if err := h.Valid(); err != nil {
		return nil, err
	}

	be.m.Lock()
	defer be.m.Unlock()

	if h.Type == restic.ConfigFile {
		h.Name = ""
	}

	debug.Log("Load %v offset %v len %v", h, offset, length)

	if offset < 0 {
		return nil, errors.New("offset is negative")
	}

	if _, ok := be.data[h]; !ok {
		return nil, errNotFound
	}

	buf := be.data[h]
	if offset > int64(len(buf)) {
		return nil, errors.New("offset beyond end of file")
	}

	buf = buf[offset:]
	if length > 0 && len(buf) > length {
		buf = buf[:length]
	}

	return ioutil.NopCloser(bytes.NewReader(buf)), nil
}

// Stat returns information about a file in the backend.
func (be *MemoryBackend) Stat(ctx context.Context, h restic.Handle) (restic.FileInfo, error) {
	be.m.Lock()
	defer be.m.Unlock()

	if err := h.Valid(); err != nil {
		return restic.FileInfo{}, err
	}

	if h.Type == restic.ConfigFile {
		h.Name = ""
	}

	debug.Log("stat %v", h)

	e, ok := be.data[h]
	if !ok {
		return restic.FileInfo{}, errNotFound
	}

	return restic.FileInfo{Size: int64(len(e))}, nil
}

// Remove deletes a file from the backend.
func (be *MemoryBackend) Remove(ctx context.Context, h restic.Handle) error {
	be.m.Lock()
	defer be.m.Unlock()

	debug.Log("Remove %v", h)

	if _, ok := be.data[h]; !ok {
		return errNotFound
	}

	delete(be.data, h)

	return nil
}

// List returns a channel which yields entries from the backend.
func (be *MemoryBackend) List(ctx context.Context, t restic.FileType) <-chan string {
	be.m.Lock()
	defer be.m.Unlock()

	ch := make(chan string)

	var ids []string
	for entry := range be.data {
		if entry.Type != t {
			continue
		}
		ids = append(ids, entry.Name)
	}

	debug.Log("list %v: %v", t, ids)

	go func() {
		defer close(ch)
		for _, id := range ids {
			select {
			case ch <- id:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// Location returns the location of the backend (RAM).
func (be *MemoryBackend) Location() string {
	return "RAM"
}

// Delete removes all data in the backend.
func (be *MemoryBackend) Delete(ctx context.Context) error {
	be.m.Lock()
	defer be.m.Unlock()

	be.data = make(memMap)
	return nil
}

// Close closes the backend.
func (be *MemoryBackend) Close() error {
	return nil
}
