package golfu

import "sync"

type RWLock[T any] struct {
	inner_lock sync.RWMutex
	value      T
}

func NewRWLock[T any](value T) *RWLock[T] {
	return &RWLock[T]{
		value: value,
	}
}

func (l *RWLock[T]) Read() T {
	l.inner_lock.RLock()
	defer l.inner_lock.RUnlock()
	return l.value
}

func (l *RWLock[T]) Write(value T) {
	l.inner_lock.Lock()
	defer l.inner_lock.Unlock()
	l.value = value
}

func (l *RWLock[T]) Update(update func(T) T) {
	l.inner_lock.Lock()
	defer l.inner_lock.Unlock()
	l.value = update(l.value)
}

func (l *RWLock[T]) WriteIf(condition func(T) bool, value T) {
	l.inner_lock.Lock()
	defer l.inner_lock.Unlock()
	if condition(l.value) {
		l.value = value
	}
}
