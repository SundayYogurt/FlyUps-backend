package cache

import (
	"context"
	"time"
)

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error // เก็บข้อมูล
	Get(ctx context.Context, key string) (string, error)                             // ดึงข้อมูล
	Del(ctx context.Context, key string) error                                       // ลบ cache
}
