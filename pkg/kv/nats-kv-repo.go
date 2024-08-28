package kv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kloudlite/kloudmeter/pkg/egob"
	"github.com/kloudlite/kloudmeter/pkg/nats"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/fx"

	"github.com/kloudlite/kloudmeter/pkg/errors"
)

type natsKVRepo[T any] struct {
	keyValue jetstream.KeyValue
}
type Value[T any] struct {
	Data      T
	ExpiresAt time.Time
}

func (v *Value[T]) isExpired() bool {
	if v.ExpiresAt.IsZero() {
		return false
	}
	return time.Since(v.ExpiresAt) > 0
}

type Entry[T any] struct {
	Key   string
	Value T
}

func (r *natsKVRepo[T]) List(c context.Context, pattern string) ([]T, error) {
	opts := []jetstream.WatchOpt{jetstream.IgnoreDeletes()}

	watcher, err := r.keyValue.Watch(c, pattern, opts...)
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	var entries []T
	for entry := range watcher.Updates() {
		if entry == nil {
			break
		}

		type Payload[K any] struct {
			Data K
		}

		var payload Payload[T]
		err := egob.Unmarshal(entry.Value(), &payload)
		if err != nil {
			return nil, errors.NewEf(err, "failed to unmarshal value")
		}

		entries = append(entries, payload.Data)
	}

	if len(entries) == 0 {
		return nil, jetstream.ErrNoKeysFound
	}

	return entries, nil
}

func (r *natsKVRepo[T]) Entries(c context.Context, pattern string) ([]Entry[T], error) {
	opts := []jetstream.WatchOpt{jetstream.IgnoreDeletes()}

	watcher, err := r.keyValue.Watch(c, pattern, opts...)
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	var entries []Entry[T]
	for entry := range watcher.Updates() {
		if entry == nil {
			break
		}

		type Payload[K any] struct {
			Data K
		}

		var payload Payload[T]

		err := egob.Unmarshal(entry.Value(), &payload)
		if err != nil {
			return nil, errors.NewEf(err, "failed to unmarshal value")
		}

		entries = append(entries, Entry[T]{
			Key:   entry.Key(),
			Value: payload.Data,
		})
	}

	if len(entries) == 0 {
		return nil, jetstream.ErrNoKeysFound
	}

	return entries, nil
}

func (r *natsKVRepo[T]) Keys(c context.Context, pattern string) ([]string, error) {
	opts := []jetstream.WatchOpt{jetstream.IgnoreDeletes(), jetstream.MetaOnly()}

	watcher, err := r.keyValue.Watch(c, pattern, opts...)
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	var keys []string
	for entry := range watcher.Updates() {
		if entry == nil {
			break
		}

		keys = append(keys, entry.Key())
	}

	if len(keys) == 0 {
		return nil, jetstream.ErrNoKeysFound
	}
	return keys, nil

}

func (r *natsKVRepo[T]) Set(c context.Context, _key string, value T) error {
	key := sanitiseKey(_key)
	v := Value[T]{
		Data: value,
	}
	b, err := egob.Marshal(v)
	if err != nil {
		return errors.NewEf(err, "failed to marshal value")
	}
	if _, err := r.keyValue.Put(c, key, b); err != nil {
		return errors.NewE(err)
	}
	return nil
}

func (r *natsKVRepo[T]) Get(c context.Context, _key string) (T, error) {
	key := sanitiseKey(_key)
	get, err := r.keyValue.Get(c, key)
	if err != nil {
		var x T
		return x, err
	}
	var value Value[T]
	err = egob.Unmarshal(get.Value(), &value)
	if value.isExpired() {
		go func() {
			if err = r.Drop(c, key); err != nil {
				fmt.Printf("unable to drop key %s", key)
			}
		}()
		return value.Data, errors.New("Key is expired")
	}
	return value.Data, err
}

func (r *natsKVRepo[T]) ErrKeyNotFound(err error) bool {
	return errors.Is(err, jetstream.ErrKeyNotFound)
}

func sanitiseKey(key string) string {
	return strings.ReplaceAll(key, ":", "-")
}

func (r *natsKVRepo[T]) SetWithExpiry(c context.Context, _key string, value T, duration time.Duration) error {
	key := sanitiseKey(_key)
	v := Value[T]{
		Data:      value,
		ExpiresAt: time.Now().Add(duration),
	}
	b, err := egob.Marshal(v)
	if err != nil {
		return errors.NewEf(err, "failed to marshal value")
	}
	if _, err := r.keyValue.Put(c, key, b); err != nil {
		return errors.NewE(err)
	}
	return nil
}

func (r *natsKVRepo[T]) Drop(c context.Context, key string) error {
	return r.keyValue.Delete(c, sanitiseKey(key))
}

func (r *natsKVRepo[T]) ErrNoRecord(err error) bool {
	return errors.Is(err, jetstream.ErrKeyNotFound)
}

func NewNatsKVRepo[T any](ctx context.Context, bucketName string, jc *nats.JetstreamClient) (Repo[T], error) {
	if value, err := jc.Jetstream.KeyValue(ctx, bucketName); err != nil && errors.Is(err, jetstream.ErrBucketNotFound) {
		_, err := jc.Jetstream.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket: bucketName,
		})
		if err != nil {
			return nil, errors.NewE(err)
		}

		value, err = jc.Jetstream.KeyValue(ctx, bucketName)
		if err != nil {
			return nil, errors.NewE(err)
		} else {
			return &natsKVRepo[T]{
				value,
			}, nil
		}
	} else if err != nil {
		return nil, errors.NewE(err)
	} else {
		return &natsKVRepo[T]{
			value,
		}, nil
	}
}

func NewNatsKvRepoFx[T any](bucketName string) fx.Option {
	return fx.Provide(func(jc *nats.JetstreamClient) (meter Repo[T], err error) {
		return NewNatsKVRepo[T](context.TODO(), bucketName, jc)
	})
}
