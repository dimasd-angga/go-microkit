package dedupe

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store is a tiny dedupe interface — at-least-once delivery means we'll
// occasionally see duplicates; this drops them.
type Store struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Store{rdb: rdb, ttl: ttl}
}

// Claim returns true if the key was claimed (i.e. first time we've seen it).
// Subsequent calls in the TTL window return false.
func (s *Store) Claim(ctx context.Context, key string) (bool, error) {
	ok, err := s.rdb.SetNX(ctx, "notif:dedupe:"+key, 1, s.ttl).Result()
	return ok, err
}
