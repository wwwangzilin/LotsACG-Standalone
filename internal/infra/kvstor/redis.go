package kvstor

import (
	"context"
	"time"

	"github.com/redis/rueidis"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"
)

type redisDB struct {
	client rueidis.Client
	prefix string
}

// Close implements KVStore.
func (r *redisDB) Close() error {
	if r.client != nil {
		r.client.Close()
	}
	return nil
}

// Set implements KVStore using Redis SET with PX for TTL when ttl > 0.
func (r *redisDB) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := msgpack.Marshal(value)
	if err != nil {
		return err
	}
	// Convert bytes to string for command builder
	sval := string(raw)
	key = r.prefix + key
	if ttl > 0 {
		// Px() expects time.Duration
		return r.client.Do(ctx, r.client.B().Set().Key(key).Value(sval).Px(ttl).Build()).Error()
	}
	return r.client.Do(ctx, r.client.B().Set().Key(key).Value(sval).Build()).Error()
}

// GetRaw implements KVStore.
func (r *redisDB) GetRaw(ctx context.Context, key string) ([]byte, error) {
	key = r.prefix + key
	rr := r.client.Do(ctx, r.client.B().Get().Key(key).Build())
	bs, err := rr.AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, errs.ErrRecordNotFound
		}
		return nil, err
	}
	return bs, nil
}

// Delete implements KVStore.
func (r *redisDB) Delete(ctx context.Context, key string) error {
	// DEL returns number of keys removed; ignore return value
	key = r.prefix + key
	return r.client.Do(ctx, r.client.B().Del().Key(key).Build()).Error()
}
