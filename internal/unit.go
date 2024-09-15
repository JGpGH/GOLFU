package internal

import (
	"sync"
	"sync/atomic"

	"github.com/JGpGH/golfu/storage"
)

type unit[T storage.Indexable] struct {
	isPersisted *atomic.Bool
	value       T
	lock        sync.RWMutex
}

type persistable[T storage.Indexable] struct {
	value       T
	isPersisted bool
}

func values[T storage.Indexable](units []*unit[T]) []T {
	var result []T
	for _, u := range units {
		result = append(result, u.Read())
	}
	return result
}

func asReadOnlyUnits[T storage.Indexable](units []*unit[T]) []storage.Readonly[T] {
	var result []storage.Readonly[T]
	for _, u := range units {
		result = append(result, u)
	}
	return result
}

func toUnits[T storage.Indexable](values []persistable[T]) []*unit[T] {
	var result []*unit[T]
	for _, v := range values {
		result = append(result, newUnit(v.value, v.isPersisted))
	}
	return result
}

func newUnit[T storage.Indexable](value T, persisted bool) *unit[T] {
	isPersisted := &atomic.Bool{}
	isPersisted.Store(persisted)
	return &unit[T]{
		value:       value,
		lock:        sync.RWMutex{},
		isPersisted: isPersisted,
	}
}

func (u *unit[T]) Read() T {
	u.lock.RLock()
	defer u.lock.RUnlock()
	return u.value
}

func (u *unit[T]) ReadOnly() storage.Readonly[T] {
	return u
}

func (u *unit[T]) Write(value T) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.value = value
	u.isPersisted.Store(false)
}

func (u *unit[T]) SetPersisted() {
	u.isPersisted.Store(true)
}

func (u *unit[T]) IsPersisted() bool {
	return u.isPersisted.Load()
}

func (u *unit[T]) Index() string {
	return u.value.Index()
}
