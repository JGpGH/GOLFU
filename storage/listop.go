package storage

import (
	"container/list"
	"sync"
	"sync/atomic"
)

type Readonly[T any] interface {
	Read() T
}

func Collect[T any](src []Readonly[T]) []T {
	var result []T
	for _, r := range src {
		result = append(result, r.Read())
	}
	return result
}

type Indexable interface {
	Index() string
}

type elementContainer[T any] struct {
	Element   *list.Element
	ReadCount *atomic.Uint32
}

type IndexedList[T Indexable] struct {
	indexed map[string]elementContainer[T]
	sorted  list.List
	lock    sync.RWMutex
}

func NewIndexedList[T Indexable]() IndexedList[T] {
	return IndexedList[T]{
		indexed: map[string]elementContainer[T]{},
		sorted:  list.List{},
		lock:    sync.RWMutex{},
	}
}

func (l *IndexedList[T]) readCount(e *list.Element) uint32 {
	if c, ok := l.indexed[e.Value.(T).Index()]; ok {
		return c.ReadCount.Load()
	}
	return 0
}

func (l *IndexedList[T]) Remove(indexes []string) int {
	count := 0
	l.lock.Lock()
	defer l.lock.Unlock()
	for _, index := range indexes {
		if c, ok := l.indexed[index]; ok {
			l.sorted.Remove(c.Element)
			delete(l.indexed, index)
			count++
		}
	}
	return count
}

func (l *IndexedList[T]) Get(indexes []string) map[string]T {
	res := make(map[string]T)
	l.lock.RLock()
	defer l.lock.RUnlock()
	for _, index := range indexes {
		if c, ok := l.indexed[index]; ok {
			v := c.Element.Value.(T)
			c.ReadCount.Add(1)
			res[index] = v
		}
	}
	return res
}

func (l *IndexedList[T]) Set(values []T) {
	l.lock.Lock()
	defer l.lock.Unlock()
	for _, v := range values {
		if c, ok := l.indexed[v.Index()]; ok {
			c.Element.Value = v
		} else {
			c := elementContainer[T]{
				Element:   l.sorted.PushBack(v),
				ReadCount: &atomic.Uint32{},
			}
			l.indexed[v.Index()] = c
		}
	}
}

func (l *IndexedList[T]) Pop(amount int) []T {
	l.lock.Lock()
	defer l.lock.Unlock()
	var result []T
	for e := l.sorted.Front(); e != nil && amount > 0; e = e.Next() {
		result = append(result, e.Value.(T))
		amount--
	}
	return result
}

// Insertion sort from the least read to the most read
func (l *IndexedList[T]) SortByReadCount() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.sorted.Len() < 2 {
		return
	}

	for e := l.sorted.Front().Next(); e != nil; e = e.Next() {
		prev := e.Prev()

		for prev != nil && l.readCount(prev) > l.readCount(e) {
			prev = prev.Prev()
		}

		if prev != nil {
			l.sorted.Remove(e)
			l.sorted.InsertAfter(e, prev)
		}
	}
}
