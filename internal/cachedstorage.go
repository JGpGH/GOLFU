package internal

import (
	"context"

	"github.com/JGpGH/golfu/internal/listop"
	"github.com/JGpGH/golfu/storage"
)

func (s *cachedStorage[T]) Start(ctx context.Context, trash storage.Trash[T]) {
	// cache storing routine for non-blocking Set
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-s.toCache:
				l := s.units.Len()
				units := toUnits(in)
				s.units.Set(units)
				s.newLength <- l + len(in)
				s.cold.Set(asReadOnlyUnits(units))
				for _, u := range units {
					u.SetPersisted()
				}
			}
		}
	}()

	// routine to evict units
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case u := <-s.newLength:
				currentLen := max(s.units.Len(), u)
				if currentLen > s.maxUnits {
					evicted := s.evict(currentLen - s.maxUnits + s.maxUnits/5) // evict 20% of the cache + everything above max
					trash.Trash(evicted)
				}
			}
		}
	}()
}

func (s *cachedStorage[T]) Set(values []T) {
	var toCache []persistable[T]
	for _, v := range values {
		toCache = append(toCache, persistable[T]{value: v, isPersisted: false})
	}
	s.toCache <- toCache
}

func (s *cachedStorage[T]) SetPersisted(values []T) {
	var toCache []persistable[T]
	for _, v := range values {
		toCache = append(toCache, persistable[T]{value: v, isPersisted: true})
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

func (s *cachedStorage[T]) evict(amount int) []T {
	if amount <= 0 {
		return []T{}
	}
	s.units.SortByReadCount()
	trashed := s.units.PopWhere(func(u *unit[T]) bool {
		return u.IsPersisted()
	}, amount)
	s.units.ClearReadCounts()
	return values(trashed)
}

type cachedStorage[T storage.Indexable] struct {
	units     listop.IndexedList[*unit[T]]
	cold      storage.ColdStorage[T]
	maxUnits  int
	ctx       context.Context
	toCache   chan []persistable[T]
	newLength chan int
}

func NewCachedStorage[T storage.Indexable](cold storage.ColdStorage[T], trash storage.Trash[T], maxUnits int) (storage.CachedStorage[T], context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cache := &cachedStorage[T]{
		units:     listop.NewIndexedList[*unit[T]](),
		cold:      cold,
		maxUnits:  maxUnits,
		ctx:       ctx,
		toCache:   make(chan []persistable[T], 100),
		newLength: make(chan int, 100),
	}
	cache.Start(ctx, trash)
	return cache, cancel
}
