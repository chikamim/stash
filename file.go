package stash

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
)

// writeFile writes a new file to the cache storage.
func writeFile(dir, key string, r io.Reader, useDeflate bool) (path string, size int64, err error) {
	path = filepath.Join(dir, key)

	f, err := os.Create(path)
	if err != nil {
		return "", 0, &FileError{dir, key, err}
	}
	defer f.Close()

	if useDeflate {
		w := NewDeflateWriter(f)
		size, err = io.Copy(w, r)
		w.Close()
	} else {
		size, err = io.Copy(f, r)
	}

	if err != nil {
		return "", 0, &FileError{dir, key, err}
	}

	return
}

func filesize(path string) (int64, error) {
	s, err := os.Stat(path)
	if err != nil {
		return 0, &FileError{path, filepath.Base(path), err}
	}
	return s.Size(), nil
}

func escape(v string) string {
	return url.QueryEscape(v)
}
