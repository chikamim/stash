package stash

import (
	"bytes"
	"container/list"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Meta struct {
	Key  string
	Size int64
	Path string
}

type Cache struct {
	dir  string // Path to storage directory
	size int64  // Total size of files allowed
	cap  int64  // Total number of files allowed

	sizeUsed int64 // Total size of files added
	capUsed  int64 // Total number of files added

	list *list.List               // List of items in cache
	m    map[string]*list.Element // Map of items in list

	useDeflate bool // Use lz4 deflate or not

	l sync.RWMutex
}

// New creates a Cache backed by dir on disk. The cache allows at most "cap" files of total size "size". If "useDeflate" is true, blobs will be compressed by lz4 for reduce disk usage.
func New(dir string, size, cap int64, useDeflate bool) (*Cache, error) {
	if !validDir(dir) {
		return nil, ErrBadDir
	}
	if size <= 0 {
		return nil, ErrBadSize
	}
	if cap <= 0 {
		return nil, ErrBadCap
	}

	dir = strings.TrimRight(dir, string(os.PathSeparator)) // Clean path to dir

	return &Cache{
		dir:        dir,
		size:       size,
		cap:        cap,
		list:       list.New(),
		m:          make(map[string]*list.Element),
		useDeflate: useDeflate,
	}, nil
}

func (c *Cache) Warmup() error {
	c.l.Lock()
	defer c.l.Unlock()

	f, err := os.Open(c.dir)
	if err != nil {
		return err
	}
	fileInfo, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return err
	}

	for _, file := range fileInfo {
		key := file.Name()
		path := filepath.Join(c.dir, key)
		c.addMeta(key, path, file.Size())
	}

	return nil
}

// Put adds a byte slice as a blob to the cache against the given key.
func (c *Cache) Put(key string, val []byte) error {
	return c.PutReader(key, bytes.NewReader(val))
}

// PutReader adds the contents of a reader as a blob to the cache against the given key.
func (c *Cache) PutReader(key string, r io.Reader) error {
	c.l.Lock()
	defer c.l.Unlock()

	path, n, err := writeFile(c.dir, escape(key), r, c.useDeflate)
	if err != nil {
		return err
	}
	if err := c.validate(path, n); err != nil { // XXX(hjr265): We should validate before storing the file.
		return err
	}
	c.addMeta(escape(key), path, n)
	return nil
}

// PutFile adds the contents of a file path as a blog to the cache. The source file will be moved or deleted.
func (c *Cache) PutFile(key, srcpath string) error {
	c.l.Lock()
	defer c.l.Unlock()

	n, err := filesize(srcpath)
	if err != nil {
		return err
	}
	path := filepath.Join(c.dir, escape(key))
	if c.useDeflate {
		w, err := os.Create(path)
		if err != nil {
			return err
		}
		defer w.Close()

		r, err := os.Open(srcpath)
		if err != nil {
			return err
		}
		defer r.Close()

		dw := NewDeflateWriter(w)
		defer dw.Close()
		n, err = io.Copy(dw, r)
		if err != nil {
			return err
		}
		os.Remove(srcpath)
	} else {
		err = os.Rename(srcpath, path)
		if err != nil {
			return err
		}
	}
	if err := c.validate(path, n); err != nil { // XXX(hjr265): We should validate before storing the file.
		return err
	}
	c.addMeta(escape(key), path, n)
	return nil
}

// Get returns a reader for a blob in the cache, or ErrNotFound otherwise.
func (c *Cache) Get(key string) (io.ReadCloser, error) {
	c.l.RLock()
	defer c.l.RUnlock()

	if item, ok := c.m[escape(key)]; ok {
		c.list.MoveToFront(item)
		path := item.Value.(*Meta).Path
		if f, err := os.Open(path); err != nil {
			return nil, err
		} else {
			if c.useDeflate {
				return NewDeflateReader(f), nil
			} else {
				return f, nil
			}
		}
	} else {
		return nil, ErrNotFound
	}
}

// Keys returns a list of keys in the cache.
func (c *Cache) Keys() []string {
	keys := make([]string, len(c.m))
	i := 0
	for item := c.list.Back(); item != nil; item = item.Prev() {
		keys[i] = item.Value.(*Meta).Key
		i++
	}
	sort.Strings(keys)
	return keys
}

// validate ensures the file satisfies the constraints of the cache.
func (c *Cache) validate(path string, n int64) error {
	if n > c.size {
		os.Remove(path) // XXX(hjr265): We should not supress this error even if it is very unlikely.
		return &FileError{c.dir, "", ErrTooLarge}
	}

	for n+c.sizeUsed > c.size {
		err := c.evictLast()
		if err != nil {
			return err
		}
	}

	if c.capUsed+1 > c.cap {
		err := c.evictLast()
		if err != nil {
			return err
		}
	}

	return nil
}

// evitLast removes the last file following the LRU policy.
func (c *Cache) evictLast() error {
	if last := c.list.Back(); last != nil {
		item := last.Value.(*Meta)
		if e := os.Remove(item.Path); e == nil {
			c.sizeUsed -= item.Size
			c.capUsed--
			delete(c.m, item.Key)
			c.list.Remove(last)
			return nil
		} else {
			return e
		}
	}

	return nil
}

// addMeta adds meta information to the cache.
func (c *Cache) addMeta(key, path string, length int64) {
	c.sizeUsed += length
	c.capUsed++
	if item, ok := c.m[key]; ok {
		c.list.Remove(item)
	}

	item := &Meta{
		Key:  key,
		Size: length,
		Path: path,
	}
	listElement := c.list.PushFront(item)
	c.m[key] = listElement
}

func validDir(dir string) bool {
	// XXX(hjr265): We need to ensure the disk is either empty, or contains a valid cache storage.

	return dir != ""
}
