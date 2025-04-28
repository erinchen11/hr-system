package interfaces

import (
	"context"
	"errors"
	"time"
)

type CacheRepository interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
}

var ErrCacheMiss = errors.New("cache: key not found")
