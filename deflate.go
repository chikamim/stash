package stash

import (
	"io"

	"github.com/pierrec/lz4"
)

type DeflateReader struct {
	r   io.Reader
	src io.ReadCloser
}

func NewDeflateReader(r io.ReadCloser) *DeflateReader {
	return &DeflateReader{src: r, r: lz4.NewReader(r)}
}

func NewDeflateWriter(w io.WriteCloser) io.WriteCloser {
	return lz4.NewWriter(w)
}

func (d *DeflateReader) Read(p []byte) (int, error) {
	return d.r.Read(p)
}

func (d *DeflateReader) Close() error {
	return d.src.Close()
}
