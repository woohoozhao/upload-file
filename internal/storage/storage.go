package storage

import (
	"context"
	"io"
)

// Storer writes uploaded files to some backing store.
//
// Implementations must drain src until io.EOF or an error, then return the
// stable identifier under which the bytes were stored (e.g. an absolute path
// for the local filesystem, or an object key for object storage).
//
// On error, implementations should clean up any partially-written artifacts
// before returning — handlers should NOT have to retry or remove themselves.
type Storer interface {
	Put(ctx context.Context, key string, src io.Reader, size int64, contentType string) (savedKey string, err error)
}
