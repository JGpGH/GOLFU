package golfu

import (
	"context"

	"github.com/JGpGH/golfu/internal"
	"github.com/JGpGH/golfu/storage"
)

func NewCachedStorage[T storage.Indexable](cold storage.ColdStorage[T], trash storage.Trash[T], maxUnits int) (storage.CachedStorage[T], context.CancelFunc) {
	return internal.NewCachedStorage[T](cold, trash, maxUnits)
}
