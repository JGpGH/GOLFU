package storage

import "context"

type ColdStorage[T Indexable] interface {
	Set([]Readonly[T]) ([]Readonly[T], error)
	Get([]string) (map[string]T, error)
}

type CachedStorage[T Indexable] interface {
	Set([]T)
	Get([]string) (map[string]T, error)
}

type cachedStorage[T Indexable] struct {
	units   IndexedList[*Unit[T]]
	cold    ColdStorage[T]
	ctx     context.Context
	cancel  context.CancelFunc
	toCache chan []Persistable[T]
}

func NewCachedStorage[T Indexable](cold ColdStorage[T]) CachedStorage[T] {
	ctx, cancel := context.WithCancel(context.Background())
	cache := &cachedStorage[T]{
		units:   NewIndexedList[*Unit[T]](),
		cold:    cold,
		ctx:     ctx,
		cancel:  cancel,
		toCache: make(chan []Persistable[T], 100),
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-cache.toCache:
				cache.units.Set(ToUnits(in))
			}
		}
	}()

	return cache
}

func (s *cachedStorage[T]) Set(values []T) {
	var toPersist []Persistable[T]
	for _, v := range values {
		toPersist = append(toPersist, Persistable[T]{value: v, isPersisted: false})
	}
	s.toCache <- toPersist
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

	toAdd := make([]T, len(persisted))
	for k, v := range persisted {
		result[k] = v
		toAdd = append(toAdd, v)
	}

	s.units.Set(ToPersistedUnits(toAdd))
	return result, nil
}

func (s *cachedStorage[T]) evict(amount int) {
	if amount <= 0 {
		return
	}

}
