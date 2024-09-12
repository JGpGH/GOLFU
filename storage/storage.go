package storage

import (
	"context"
	"golfu/listop"
)

type ColdStorage[T listop.Indexable] interface {
	Set([]listop.Readonly[T]) ([]listop.Readonly[T], error)
	Get([]string) (map[string]T, error)
}

type CachedStorage[T listop.Indexable] interface {
	Set([]T)
	Get([]string) (map[string]T, error)
}

type cachedStorage[T listop.Indexable] struct {
	units     listop.IndexedList[*Unit[T]]
	cold      ColdStorage[T]
	maxUnits  int
	ctx       context.Context
	cancel    context.CancelFunc
	toCache   chan []Persistable[T]
	newLength chan int
}

func NewCachedStorage[T listop.Indexable](cold ColdStorage[T], maxUnits int) CachedStorage[T] {
	ctx, cancel := context.WithCancel(context.Background())
	cache := &cachedStorage[T]{
		units:     listop.NewIndexedList[*Unit[T]](),
		cold:      cold,
		maxUnits:  maxUnits,
		ctx:       ctx,
		cancel:    cancel,
		toCache:   make(chan []Persistable[T], 100),
		newLength: make(chan int, 100),
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-cache.toCache:
				l := cache.units.Len()
				cache.units.Set(ToUnits(in))
				cache.newLength <- l + len(in)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case u := <-cache.newLength:
				if u > cache.maxUnits {
					cache.evict(u - cache.maxUnits + cache.maxUnits/5) // evict 20% of the cache + everything above max
				}
			}
		}
	}()

	return cache
}

func (s *cachedStorage[T]) Set(values []T) {
	var toCache []Persistable[T]
	for _, v := range values {
		toCache = append(toCache, Persistable[T]{value: v, isPersisted: false})
	}
	s.toCache <- toCache
}

func (s *cachedStorage[T]) SetPersisted(values []T) {
	var toCache []Persistable[T]
	for _, v := range values {
		toCache = append(toCache, Persistable[T]{value: v, isPersisted: true})
	}
	s.toCache <- toCache
}

func (s *cachedStorage[T]) Get(indexes []string) (map[string]T, error) {
	var result = make(map[string]T)
	var toFetch []string
	cached := s.units.Get(indexes)
	for _, c := range indexes {
		if u, ok := cached[c]; ok {
			result[c] = u.Read()
		} else {
			toFetch = append(toFetch, c)
		}
	}

	if len(toFetch) == 0 {
		return result, nil
	}

	persisted, err := s.cold.Get(toFetch)
	if err != nil {
		return nil, err
	}

	toCache := make([]T, len(persisted))
	for k, v := range persisted {
		result[k] = v
		toCache = append(toCache, v)
	}

	s.Set(toCache)
	return result, nil
}

func (s *cachedStorage[T]) evict(amount int) {
	if amount <= 0 {
		return
	}
	s.units.SortByReadCount()
	s.units.PopWhere(func(u *Unit[T]) bool {
		return !u.IsPersisted()
	}, amount)
}
