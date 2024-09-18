package golfu

import (
	"context"

	"github.com/JGpGH/golfu/internal"
	"github.com/JGpGH/golfu/storage"
)

func NewCachedStorage[T storage.Indexable](ctx context.Context, cold storage.ColdStorage[T], trash storage.Trash[T], maxUnits int) storage.CachedStorage[T] {
	return internal.NewCachedStorage[T](ctx, cold, trash, maxUnits)
}
