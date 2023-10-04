package odfi

import (
	"crypto/sha256"
	gohash "hash"
	"io"
)

type HashFilter struct {
	wrappedReader io.Reader
	hash          gohash.Hash
}

func NewHashFilter(wrappedReader io.Reader) *HashFilter {
	return &HashFilter{
		wrappedReader: wrappedReader,
		hash:          sha256.New(),
	}
}

func (h *HashFilter) Read(p []byte) (n int, err error) {
	n, err = h.wrappedReader.Read(p)
	if err != nil {
		return n, err
	}
	_, err = h.hash.Write(p[:n])
	return n, err
}

func (h *HashFilter) Sum() []byte {
	return h.hash.Sum(nil)
}
