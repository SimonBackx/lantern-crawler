package crawler

import (
	"errors"
	"io"
)

type CountingReader struct {
	Reader io.Reader
	Size   int64
	Limit  int64
}

func NewCountingReader(reader io.Reader, limit int64) *CountingReader {
	return &CountingReader{
		Reader: reader,
		Size:   0,
		Limit:  limit,
	}
}

func (r *CountingReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Size += int64(n)

	if r.Size > r.Limit && err == nil {
		err = errors.New("Reader reached maximum bytes!")
	}
	return n, err
}
