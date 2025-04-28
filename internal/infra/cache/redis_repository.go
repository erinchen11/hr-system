package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces" 
	"github.com/redis/go-redis/v9"
)

type redisCacheRepository struct {
	client *redis.Client
}

// RedisConfig 和 InitializeCache 保持不變
type RedisConfig struct {
	Addr   string
	Passwd string
	DB     int
}

func InitializeCache(cfg RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Passwd,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return rdb, nil
}

// *** 構造函數名稱和返回類型 ***
// NewRedisCacheRepository 包裝成 CacheRepository 介面
func NewRedisCacheRepository(client *redis.Client) interfaces.CacheRepository { // <-- 返回 interfaces.CacheRepository
	return &redisCacheRepository{ // <-- 返回 後的 struct
		client: client,
	}
}

// Get 從 Redis 取資料
// ***  方法接收者和錯誤返回 ***
func (r *redisCacheRepository) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// ***  : 返回標準的未找到錯誤 ***
			return interfaces.ErrCacheMiss
		}
		// 包裝其他 redis 錯誤
		return fmt.Errorf("redis get failed for key %s: %w", key, err)
	}

	// JSON Unmarshal 部分保持不變
	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return fmt.Errorf("redis unmarshal failed for key %s: %w", key, err)
	}
	return nil
}

// Set 將資料存進 Redis 並指定過期時間
func (r *redisCacheRepository) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redis marshal failed for key %s: %w", key, err)
	}
	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set failed for key %s: %w", key, err)
	}
	return nil
}

// Delete 刪除 Redis key
// ***  方法簽名和內部調用以匹配介面 ***
func (r *redisCacheRepository) Delete(ctx context.Context, keys ...string) error { // <-- 改為接收 ...string
	if len(keys) == 0 {
		return nil // 如果沒有傳入 keys，直接返回 nil
	}
	// 將接收到的可變參數 keys 直接傳遞給 redis client 的 Del 方法
	err := r.client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("redis delete failed for keys %v: %w", keys, err)
	}
	return nil
}
