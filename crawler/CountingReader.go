package crawler

import (
	"errors"
	"io"
)

type CountingReader struct {
	Reader io.Reader
	Size   int
	Limit  int
}

func NewCountingReader(reader io.Reader, limit int) *CountingReader {
	return &CountingReader{
		Reader: reader,
		Size:   0,
		Limit:  limit,
	}
}

func (r *CountingReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Size += n

	if r.Size > r.Limit && err == nil {
		err = errors.New("Reader reached maximum bytes!")
	}
	return n, err
}

type PositionReader struct {
	Reader   io.Reader
	Position int
}

func NewPositionReader(reader io.Reader) *PositionReader {
	return &PositionReader{
		Reader: reader,
	}
}

func (r *PositionReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Position += n
	return n, err
}
