package storage

import (
	"sync"
	"sync/atomic"
)

type Unit[T Indexable] struct {
	isPersisted *atomic.Bool
	value       T
	lock        sync.RWMutex
}

type Persistable[T Indexable] struct {
	value       T
	isPersisted bool
}

func ToUnpersistedUnits[T Indexable](values []T) []*Unit[T] {
	var result []*Unit[T]
	for _, v := range values {
		result = append(result, NewUnit(v, false))
	}
	return result
}

func ToPersistedUnits[T Indexable](values []T) []*Unit[T] {
	var result []*Unit[T]
	for _, v := range values {
		result = append(result, NewUnit(v, true))
	}
	return result
}

func ToUnits[T Indexable](values []Persistable[T]) []*Unit[T] {
	var result []*Unit[T]
	for _, v := range values {
		result = append(result, NewUnit(v.value, v.isPersisted))
	}
	return result
}

func NewUnit[T Indexable](value T, persisted bool) *Unit[T] {
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
