package kvstor

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/shared/errs"
	"go.etcd.io/bbolt"
)

type bboltItem[T any] struct {
	Value     T
	CreatedAt time.Time
	ExpiresAt int64
}

func newbboltItem[T any](value T) bboltItem[T] {
	return bboltItem[T]{
		Value:     value,
		CreatedAt: time.Now(),
	}
}

func newbboltItemWithTTL[T any](value T, ttl time.Duration) bboltItem[T] {
	item := newbboltItem(value)
	if ttl > 0 {
		item.ExpiresAt = item.CreatedAt.Add(ttl).UnixNano()
	}
	return item
}

type bboltDB struct {
	db             *bbolt.DB
	stop           chan struct{}
	bucket         string
	ttlBucket      string
	ttlBatchLimit  int
	ttlSweepPeriod time.Duration
}

// Close implements KVStore.
func (b *bboltDB) Close() error {
	return b.db.Close()
}

// Delete implements KVStore.
func (b *bboltDB) Delete(ctx context.Context, key string) error {
	db := b.db
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(b.bucket))
		if bucket == nil {
			return errs.ErrRecordNotFound
		}
		raw := bucket.Get([]byte(key))
		if raw != nil {
			var existing bboltItem[[]byte]
			if err := msgpack.Unmarshal(raw, &existing); err == nil && existing.ExpiresAt > 0 {
				ttlBucket := tx.Bucket([]byte(b.ttlBucket))
				if ttlBucket != nil {
					_ = ttlBucket.Delete(b.encodeTTLKey(existing.ExpiresAt, key))
				}
			}
		}
		return bucket.Delete([]byte(key))
	})
}

// GetRaw implements KVStore.
func (b *bboltDB) GetRaw(ctx context.Context, key string) ([]byte, error) {
	db := b.db
	var val []byte
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(b.bucket))
		if bucket == nil {
			return errs.ErrRecordNotFound
		}
		val = bucket.Get([]byte(key))
		if val == nil {
			return errs.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var result bboltItem[[]byte]
	if err := msgpack.Unmarshal(val, &result); err != nil {
		return nil, err
	}
	if result.ExpiresAt > 0 && time.Now().UnixNano() > result.ExpiresAt {
		b.deleteExpired(key, result.ExpiresAt)
		return nil, errs.ErrRecordNotFound
	}
	return result.Value, nil
}

// SetWithTTL implements KVStore.
func (b *bboltDB) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	db := b.db
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(b.bucket))
		if err != nil {
			return err
		}
		ttlBucket, err := tx.CreateBucketIfNotExists([]byte(b.ttlBucket))
		if err != nil {
			return err
		}

		if prev := bucket.Get([]byte(key)); prev != nil {
			var existing bboltItem[[]byte]
			if err := msgpack.Unmarshal(prev, &existing); err == nil && existing.ExpiresAt > 0 {
				if err := ttlBucket.Delete(b.encodeTTLKey(existing.ExpiresAt, key)); err != nil {
					return err
				}
			}
		}

		// 先把 value 编码成字节, 再存进 bboltItem[[]byte] (带 TTL 元信息),
		// 这样 GetRaw 拿到的原始字节可直接由调用方 msgpack 解码到具体类型。
		valBytes, err := msgpack.Marshal(value)
		if err != nil {
			return err
		}
		entry := newbboltItemWithTTL(valBytes, ttl)
		val, err := msgpack.Marshal(entry)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(key), val); err != nil {
			return err
		}
		if entry.ExpiresAt > 0 {
			if err := ttlBucket.Put(b.encodeTTLKey(entry.ExpiresAt, key), []byte{1}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *bboltDB) deleteExpired(key string, expiresAt int64) error {
	db := b.db
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(b.bucket))
		if bucket == nil {
			return nil
		}
		ttlBucket := tx.Bucket([]byte(b.ttlBucket))
		if ttlBucket != nil {
			ttlBucket.Delete(b.encodeTTLKey(expiresAt, key))
		}
		return bucket.Delete([]byte(key))
	})
}

func (b *bboltDB) sweepExpired() error {
	db := b.db
	now := time.Now().UnixNano()
	return db.Update(func(tx *bbolt.Tx) error {
		ttlBucket := tx.Bucket([]byte(b.ttlBucket))
		if ttlBucket == nil {
			return nil
		}
		bucket := tx.Bucket([]byte(b.bucket))
		if bucket == nil {
			return nil
		}
		cursor := ttlBucket.Cursor()
		processed := 0
		for k, _ := cursor.First(); k != nil && processed < b.ttlBatchLimit; k, _ = cursor.Next() {
			if len(k) < 8 {
				continue
			}
			expiresAt := int64(binary.BigEndian.Uint64(k[:8]))
			if expiresAt > now {
				break
			}
			rawKey := k[8:]
			if err := bucket.Delete(rawKey); err != nil {
				return err
			}
			if err := cursor.Delete(); err != nil {
				return err
			}
			processed++
		}
		return nil
	})
}

func (b *bboltDB) startTTLReaper() {
	go func() {
		ticker := time.NewTicker(b.ttlSweepPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.sweepExpired()
			case <-b.stop:
				return
			}
		}
	}()
}

func (b *bboltDB) encodeTTLKey(expiresAt int64, key string) []byte {
	buf := make([]byte, 8+len(key))
	binary.BigEndian.PutUint64(buf[:8], uint64(expiresAt))
	copy(buf[8:], key)
	return buf
}
