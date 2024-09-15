package storage

import (
	"sync"
	"sync/atomic"

	"github.com/JGpGH/GOLFU/listop"
)

type Unit[T listop.Indexable] struct {
	isPersisted *atomic.Bool
	value       T
	lock        sync.RWMutex
}

type Persistable[T listop.Indexable] struct {
	value       T
	isPersisted bool
}

func ToUnpersistedUnits[T listop.Indexable](values []T) []*Unit[T] {
	var result []*Unit[T]
	for _, v := range values {
		result = append(result, NewUnit(v, false))
	}
	return result
}

func Values[T listop.Indexable](units []*Unit[T]) []T {
	var result []T
	for _, u := range units {
		result = append(result, u.Read())
	}
	return result
}

func AsReadOnlyUnits[T listop.Indexable](units []*Unit[T]) []listop.Readonly[T] {
	var result []listop.Readonly[T]
	for _, u := range units {
		result = append(result, u)
	}
	return result
}

func ToPersistedUnits[T listop.Indexable](values []T) []*Unit[T] {
	var result []*Unit[T]
	for _, v := range values {
		result = append(result, NewUnit(v, true))
	}
	return result
}

func ToUnits[T listop.Indexable](values []Persistable[T]) []*Unit[T] {
	var result []*Unit[T]
	for _, v := range values {
		result = append(result, NewUnit(v.value, v.isPersisted))
	}
	return result
}

func NewUnit[T listop.Indexable](value T, persisted bool) *Unit[T] {
	isPersisted := &atomic.Bool{}
	isPersisted.Store(persisted)
	return &Unit[T]{
		value:       value,
		lock:        sync.RWMutex{},
		isPersisted: isPersisted,
	}
}

func (u *Unit[T]) Read() T {
	u.lock.RLock()
	defer u.lock.RUnlock()
	return u.value
}

func (u *Unit[T]) ReadOnly() listop.Readonly[T] {
	return u
}

func (u *Unit[T]) Write(value T) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.value = value
	u.isPersisted.Store(false)
}

func (u *Unit[T]) SetPersisted() {
	u.isPersisted.Store(true)
}

func (u *Unit[T]) IsPersisted() bool {
	return u.isPersisted.Load()
}

func (u *Unit[T]) Index() string {
	return u.value.Index()
}
