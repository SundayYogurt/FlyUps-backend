package redis

import (
	"context"
	"flyup/internal/port/cache"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	RDB *redis.Client
}

var _ cache.Cache = (*Client)(nil)

// NewRedisClient สร้าง connection ไป Redis
func NewRedisClient(addr, password string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,            // คือที่อยู่ Redis server
		Password:     password,        // รหัสผ่านของ Redis server
		DB:           db,              // Redis มีหลาย database (0-15)
		PoolSize:     10,              //PoolSize: max connection
		MinIdleConns: 5,               //MinIdleConns: connection ที่เตรียมไว้ ช่วย performance ตอนมี request เยอะ
		DialTimeout:  5 * time.Second, // ป้องกันค้าง:
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	return &Client{RDB: rdb} // เอา redis client ใส่ใน struct
}

// Ping เช็คว่า Redis ใช้งานได้
func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // ถ้า Redis ไม่ตอบใน 2 วิ ให้ cancel
	defer cancel()

	return c.RDB.Ping(ctx).Err() // ยิงคำสั่ง error
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.RDB.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.RDB.Get(ctx, key).Result()
}

func (c *Client) Del(ctx context.Context, key string) error {
	return c.RDB.Del(ctx, key).Err()
}

func (c *Client) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return c.RDB.SetNX(ctx, key, value, ttl).Result()
}
