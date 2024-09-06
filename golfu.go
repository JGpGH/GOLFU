package golfu

import "sync/atomic"

type Unit[T any] struct {
	readCount   *atomic.Int32
	isPersisted *atomic.Bool
	value       *RWLock[T]
}

func NewUnit[T any](value T) *Unit[T] {
	readCount := &atomic.Int32{}
	readCount.Store(0)
	isPersisted := &atomic.Bool{}
	isPersisted.Store(false)
	return &Unit[T]{
		value:       NewRWLock(value),
		readCount:   readCount,
		isPersisted: isPersisted,
	}
}

func (u *Unit[T]) Read() T {
	u.readCount.Add(1)
	return u.value.Read()
}
