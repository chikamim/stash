package stash

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pierrec/lz4"
)

var storageDir string

// clearStorage empties the temporary storage directory
func clearStorage() {
	err := os.RemoveAll(storageDir)
	if err != nil {
		panic(err)
	}

	err = os.Mkdir(storageDir, 0777)
	if err != nil {
		panic(err)
	}
}

func TestNew(t *testing.T) {
	for i, c := range []struct {
		dir string
		sz  int64
		c   int64
		err error
	}{
		{
			dir: "",
			sz:  2048,
			c:   4,
			err: ErrBadDir,
		},
		{
			dir: storageDir,
			sz:  0,
			c:   0,
			err: ErrBadSize,
		},
		{
			dir: storageDir,
			sz:  2048,
			c:   0,
			err: ErrBadCap,
		},
	} {
		clearStorage()

		_, err := New(c.dir, c.sz, c.c, false)
		if err != c.err {
			t.Fatalf("#%d: Expected err == %q, got %q", i+1, c.err, err)
		}
	}
}

func TestCachePut(t *testing.T) {
	clearStorage()

	s, err := New(storageDir, 2048000, 40, false)
	catch(err)
	for k, b := range blobs {
		err := s.Put(k, b)
		catch(err)
	}

	for k, b := range blobs {
		path := filepath.Join(storageDir, escape(k))
		v, err := ioutil.ReadFile(path)
		catch(err)
		if !bytes.Equal(b, v) {
			t.Fatalf("Expected v == %q, got %q", b, v)
		}
	}
}

func TestCachePutFile(t *testing.T) {
	clearStorage()

	filename := "putfile"
	k := "file"
	b := []byte("abcdefgh")

	s, err := New(storageDir, 2048000, 40, false)
	catch(err)
	f, err := os.Create(filename)
	catch(err)
	defer os.Remove(filename)
	f.Write(b)
	err = s.PutFile(k, filename)
	catch(err)

	path := filepath.Join(storageDir, escape(k))
	v, err := ioutil.ReadFile(path)
	catch(err)
	if !bytes.Equal(b, v) {
		t.Fatalf("Expected v == %q, got %q", b, v)
	}
}

func TestCachePutFileDeflate(t *testing.T) {
	//TODO:
}

func TestCachePutDeflate(t *testing.T) {
	clearStorage()

	key := "key"
	value := []byte("value")

	s, err := New(storageDir, 2048000, 40, true)
	catch(err)
	s.Put(key, value)

	path := filepath.Join(storageDir, escape(key))
	f, _ := os.Open(path)
	defer f.Close()

	r := lz4.NewReader(f)
	got, _ := ioutil.ReadAll(r)

	if !bytes.Equal(got, value) {
		t.Fatalf("Expected v == %q, got %q", value, got)
	}
}

func TestCacheGetDeflate(t *testing.T) {
	clearStorage()

	key := "key"
	value := []byte("value")

	s, err := New(storageDir, 2048000, 40, true)
	catch(err)
	s.Put(key, value)

	path := filepath.Join(storageDir, escape(key))
	f, _ := os.Create(path)
	defer f.Close()

	w := lz4.NewWriter(f)
	w.Write(value)
	w.Close()

	r, _ := s.Get(key)
	got, _ := ioutil.ReadAll(r)
	if !bytes.Equal(got, value) {
		t.Fatalf("Expected v == %q, got %q", value, got)
	}
}

func TestWarmup(t *testing.T) {
	clearStorage()

	s, err := New(storageDir, 2048000, 40, false)
	catch(err)
	for k, b := range blobs {
		path := filepath.Join(storageDir, escape(k))
		err := ioutil.WriteFile(path, b, 0666)
		catch(err)
	}

	s.Warmup()

	for k, b := range blobs {
		r, err := s.Get(escape(k))
		catch(err)
		v, err := ioutil.ReadAll(r)
		catch(err)
		if !bytes.Equal(b, v) {
			t.Fatalf("Expected v == %q, got %q", b, v)
		}
	}
}

func TestSizeEviction(t *testing.T) {
	clearStorage()

	s, err := New(storageDir, 10, 40, false)
	catch(err)

	err = s.Put("a", []byte("abcdefgh"))
	catch(err)
	err = s.Put("b", []byte("ij"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"a", "b"})

	err = s.Put("c", []byte("k"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"b", "c"})

	err = s.Put("d", []byte("l"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"b", "c", "d"})

	err = s.Put("e", []byte("m"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"b", "c", "d", "e"})

	err = s.Put("f", []byte("nopqrstuvw"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"f"})
}

func TestCapEviction(t *testing.T) {
	clearStorage()

	s, err := New(storageDir, 2048, 3, false)
	catch(err)

	err = s.Put("a", []byte("abcdefg"))
	catch(err)
	err = s.Put("b", []byte("hi"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"a", "b"})

	err = s.Put("c", []byte("k"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"a", "b", "c"})

	err = s.Put("d", []byte("l"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"b", "c", "d"})

	err = s.Put("e", []byte("m"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"c", "d", "e"})

	err = s.Put("f", []byte("nopqrstuv"))
	catch(err)
	assertKeys(t, s.Keys(), []string{"d", "e", "f"})
}

func TestMain(m *testing.M) {
	// Create a temporary storage directory for tests
	name, err := ioutil.TempDir("", "stash-")
	if err != nil {
		log.Fatal(err)
	}
	storageDir = name
	defer os.RemoveAll(name)

	os.Exit(m.Run())
}

func assertKeys(t *testing.T, keys []string, expected []string) {
	if len(keys) != len(expected) {
		t.Fatalf("Expected %d key(s), got %d", len(expected), len(keys))
	}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("Expected keys == %q, got %q", expected, keys)
	}
}

func catch(err error) {
	if err != nil {
		panic(err)
	}
}

var blobs = map[string][]byte{
	"gopher":      []byte(`The Go gopher is an iconic mascot and one of the most distinctive features of the Go project. In this post we'll talk about its origins, evolution, and behavior.`),
	"io/ioutil":   []byte(`Package ioutil implements some I/O utility functions.`),
	"testing.go":  []byte(`Package testing provides support for automated testing of Go packages.`),
	"empty.txt":   []byte(``),
	"hello-world": []byte(`Hello, world!`),
	"null":        []byte{0},
}
