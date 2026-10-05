package key

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFlatKeyBuilder(t *testing.T) {
	t.Run("no dir prefix", func(t *testing.T) {
		before := time.Now().UnixNano()
		k := FlatKeyBuilder{}.Build("photo", ".jpg")
		after := time.Now().UnixNano()

		if !strings.HasPrefix(k, "photo_") {
			t.Errorf("key should start with 'photo_', got %q", k)
		}
		if !strings.HasSuffix(k, ".jpg") {
			t.Errorf("key should end with '.jpg', got %q", k)
		}
		if strings.Contains(k, "/") {
			t.Errorf("FlatKeyBuilder without Dir should not contain '/', got %q", k)
		}

		tsStr := strings.TrimPrefix(k, "photo_")
		tsStr = strings.TrimSuffix(tsStr, ".jpg")
		parts := strings.Split(tsStr, "_")
		if len(parts) != 2 {
			t.Fatalf("expected 2 underscore-separated parts in %q, got %d", tsStr, len(parts))
		}
		var ts int64
		if _, err := fmt.Sscanf(parts[0], "%d", &ts); err != nil {
			t.Fatalf("failed to parse nano timestamp: %v", err)
		}
		if ts < before || ts > after {
			t.Errorf("timestamp %d not in [%d, %d]", ts, before, after)
		}
	})

	t.Run("with dir prefix", func(t *testing.T) {
		k := FlatKeyBuilder{Dir: "uploads"}.Build("doc", ".pdf")
		if !strings.HasPrefix(k, "uploads/doc_") {
			t.Errorf("key should start with 'uploads/doc_', got %q", k)
		}
		if !strings.HasSuffix(k, ".pdf") {
			t.Errorf("key should end with '.pdf', got %q", k)
		}
	})

	t.Run("collision avoidance", func(t *testing.T) {
		b := FlatKeyBuilder{}
		seen := map[string]struct{}{}
		for i := 0; i < 100; i++ {
			k := b.Build("same", ".jpg")
			if _, dup := seen[k]; dup {
				t.Fatalf("collision on iteration %d: key=%q", i, k)
			}
			seen[k] = struct{}{}
		}
	})
}

func TestShardedKeyBuilder(t *testing.T) {
	b := ShardedKeyBuilder{Prefix: "avatars", Layout: "2006/01/02"}

	t.Run("prefix and date layout", func(t *testing.T) {
		k := b.Build("photo", ".jpg")
		if !strings.HasPrefix(k, "avatars/") {
			t.Errorf("key should start with 'avatars/', got %q", k)
		}
		if !strings.HasSuffix(k, ".jpg") {
			t.Errorf("key should end with '.jpg', got %q", k)
		}
		if _, err := time.Parse("2006/01/02", strings.TrimPrefix(k, "avatars/")[:10]); err != nil {
			t.Errorf("expected YYYY/MM/DD segment after prefix, got key %q: %v", k, err)
		}
	})

	t.Run("default layout when empty", func(t *testing.T) {
		k := ShardedKeyBuilder{Prefix: "p"}.Build("a", ".bin")
		if strings.Count(k, "/") < 3 {
			t.Errorf("expected at least 3 '/' in sharded key, got %q", k)
		}
	})
}
