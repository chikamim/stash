package stash

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
)

// writeFile writes a new file to the cache storage.
func writeFile(dir, key string, r io.Reader) (string, int64, error) {
	name := escape(key)
	path := filepath.Join(dir, name)

	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return "", 0, &FileError{dir, key, err}
	}
	n, err := io.Copy(f, r)
	if err != nil {
		return "", 0, &FileError{dir, key, err}
	}

	return path, n, nil
}

func filesize(path string) (int64, error) {
	s, err := os.Stat(path)
	if err != nil {
		return 0, &FileError{path, escape(filepath.Base(path)), err}
	}
	return s.Size(), nil
}

func escape(v string) string {
	return url.QueryEscape(v)
}
