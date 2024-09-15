package internal_test

import (
	"context"
	"testing"
	"time"

	"github.com/JGpGH/golfu/internal"
	"github.com/JGpGH/golfu/storage"
)

type TestColdStorage[T storage.Indexable] struct {
	setted  chan T
	deleted chan T
	inner   map[string]T
}

func NewTestColdStorage[T storage.Indexable]() *TestColdStorage[T] {
	return &TestColdStorage[T]{
		setted:  make(chan T, 100),
		deleted: make(chan T, 100),
		inner:   make(map[string]T),
	}
}

func (tcs *TestColdStorage[T]) Set(ins []storage.Readonly[T]) error {
	for _, in := range ins {
		tcs.setted <- in.Read()
	}
	return nil
}

func (tcs *TestColdStorage[T]) Get(keys []string) (map[string]T, error) {
	res := make(map[string]T)
	for _, key := range keys {
		res[key] = tcs.inner[key]
	}
	return tcs.inner, nil
}

func (tcs *TestColdStorage[T]) Trash(ins []T) {
	for _, in := range ins {
		tcs.deleted <- in
	}
}

func (tcs *TestColdStorage[T]) CollectSetted(ctx context.Context, max int) []T {
	res := make([]T, 0)
	for {
		select {
		case <-ctx.Done():
			return res
		case in := <-tcs.setted:
			res = append(res, in)
			if len(res) >= max {
				return res
			}
		}
	}
}

func (tcs *TestColdStorage[T]) CollectDeleted(ctx context.Context, max int) []T {
	res := make([]T, 0)
	for {
		select {
		case <-ctx.Done():
			return res
		case in := <-tcs.deleted:
			res = append(res, in)
			if len(res) >= max {
				return res
			}
		}
	}
}

func TestStorageGetThenSet(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache, stop := internal.NewCachedStorage(cold, cold, 10)
	defer stop()
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 3)
	cancel()
	res, err := cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	if res["1"].Value != 1 || res["2"].Value != 2 || res["3"].Value != 3 {
		t.Error("Get failed")
	}
}

func TestStorageEvictsSortedByRead(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache, cancel := internal.NewCachedStorage(cold, cold, 4)
	defer cancel()
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 3)
	cancel()
	_, err := cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("5", 0),
		storage.NewIndexed("6", 0),
	})
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 1)
	cancel()
	if len(r) != 1 {
		t.Error("Eviction failed")
	}
	for _, in := range r {
		if in.Value != 0 {
			t.Error("Eviction of higher ranked values detected")
		}
	}
}

func TestStorageEvictsOldest(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache, cancel := internal.NewCachedStorage(cold, cold, 4)
	defer cancel()
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 3)
	cancel()
	_, err := cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("5", 0),
		storage.NewIndexed("6", 0),
	})
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 2)
	cancel()
	_, err = cache.Get([]string{"5", "6"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get([]string{"5", "6"})
	if err != nil {
		t.Error(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 1)
	cancel()
	if len(r) != 1 {
		t.Error("Eviction failed")
	}
	for _, in := range r {
		if in.Value == 0 {
			t.Error("Eviction of newer values detected")
		}
	}
}

func TestStorageEvictsUntil20PercentUnderMax(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache, cancel := internal.NewCachedStorage(cold, cold, 10)
	defer cancel()
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
		storage.NewIndexed("4", 4),
		storage.NewIndexed("5", 5),
		storage.NewIndexed("6", 6),
		storage.NewIndexed("7", 7),
		storage.NewIndexed("8", 8),
		storage.NewIndexed("9", 9),
		storage.NewIndexed("10", 10),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 10)
	cancel()
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("11", 11),
	})
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 3)
	cancel()
	if len(r) != 3 {
		t.Error("Eviction of 20% under max failed")
	}
}
