package key

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// KeyBuilder combines a renamed base + extension into a final storage key.
// Implementations decide on directory layout, collision avoidance, and
// partitioning strategy.
type KeyBuilder interface {
	Build(baseName, ext string) string
}

// FlatKeyBuilder produces keys like "<baseName>_<unix-nano>_<rand>.<ext>"
// under Dir. Dir is prepended when non-empty, so pass "" for root.
//
// Uses nanosecond precision + a random suffix so concurrent uploads with
// the same filename never collide.
type FlatKeyBuilder struct {
	Dir string
}

func (b FlatKeyBuilder) Build(baseName, ext string) string {
	if b.Dir == "" {
		return fmt.Sprintf("%s_%d_%d%s", baseName, time.Now().UnixNano(), rand.IntN(1<<16), ext)
	}
	return fmt.Sprintf("%s/%s_%d_%d%s", b.Dir, baseName, time.Now().UnixNano(), rand.IntN(1<<16), ext)
}

// ShardedKeyBuilder produces date-partitioned keys like
// "<Prefix>/YYYY/MM/DD/<baseName>_<rand>.<ext>". Date partitioning is useful
// with OSS lifecycle rules that archive or expire objects by prefix/date.
//
// Layout follows Go's reference time format (e.g. "2006/01/02").
type ShardedKeyBuilder struct {
	Prefix string
	Layout string
}

func (b ShardedKeyBuilder) Build(baseName, ext string) string {
	layout := b.Layout
	if layout == "" {
		layout = "2006/01/02"
	}
	return fmt.Sprintf("%s/%s/%s_%d%s",
		b.Prefix, time.Now().Format(layout),
		baseName, rand.IntN(1<<16), ext)
}
