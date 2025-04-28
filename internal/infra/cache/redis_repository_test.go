package cache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces" 
	"github.com/go-redis/redismock/v9"                    
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert" 
)

// 用於測試的簡單結構體
type testStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// 測試 Get 方法
func TestRedisCacheRepository_Get(t *testing.T) {
	ctx := context.Background()
	db, mock := redismock.NewClientMock() // <-- 創建 mock client
	repo := NewRedisCacheRepository(db)   // <-- 使用 mock client 創建 repository

	key := "test:get"
	data := testStruct{Name: "Alice", Age: 30}
	jsonData, _ := json.Marshal(data) // 預期 Redis 返回的 JSON 字串

	// 測試案例 1: Cache Hit (成功找到 Key)
	t.Run("Cache Hit", func(t *testing.T) {
		var dest testStruct
		mock.ExpectGet(key).SetVal(string(jsonData)) // <-- 設置 Get 的預期行為和返回值

		err := repo.Get(ctx, key, &dest)

		assert.NoError(t, err)                        // 斷言沒有錯誤
		assert.Equal(t, data, dest)                   // 斷言反序列化的結果正確
		assert.NoError(t, mock.ExpectationsWereMet()) // 斷言所有 mock 預期都被滿足
	})

	// 測試案例 2: Cache Miss (Key 不存在)
	t.Run("Cache Miss", func(t *testing.T) {
		var dest testStruct
		mock.ExpectGet(key).SetErr(redis.Nil) // <-- 設置 Get 返回 redis.Nil 錯誤

		err := repo.Get(ctx, key, &dest)

		assert.ErrorIs(t, err, interfaces.ErrCacheMiss) // <-- 斷言返回的是定義的 ErrCacheMiss
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// 測試案例 3: Redis 錯誤 (非 Nil 錯誤)
	t.Run("Redis Error", func(t *testing.T) {
		var dest testStruct
		expectedErr := errors.New("some redis connection error")
		mock.ExpectGet(key).SetErr(expectedErr) // <-- 設置 Get 返回一個自訂錯誤

		err := repo.Get(ctx, key, &dest)

		assert.Error(t, err)                                // 斷言有錯誤發生
		assert.NotErrorIs(t, err, interfaces.ErrCacheMiss)  // 斷言錯誤不是 ErrCacheMiss
		assert.Contains(t, err.Error(), "redis get failed") // 檢查錯誤訊息是否被包裝
		assert.ErrorIs(t, err, expectedErr)                 // 檢查原始錯誤是否被包含 (使用 %w 包裝的好處)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// 測試案例 4: JSON Unmarshal 錯誤
	t.Run("Unmarshal Error", func(t *testing.T) {
		var dest testStruct
		mock.ExpectGet(key).SetVal("invalid json {") // <-- 設置 Get 返回一個無效的 JSON 字串

		err := repo.Get(ctx, key, &dest)

		assert.Error(t, err)                                      // 斷言有錯誤發生
		assert.NotErrorIs(t, err, interfaces.ErrCacheMiss)        // 斷言錯誤不是 ErrCacheMiss
		assert.Contains(t, err.Error(), "redis unmarshal failed") // 檢查錯誤訊息
		// 檢查是否為 json.SyntaxError，或者更簡單地檢查錯誤訊息
		var syntaxError *json.SyntaxError
		assert.ErrorAs(t, err, &syntaxError) // 斷言錯誤鏈中包含 json.SyntaxError
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// 測試 Set 方法
func TestRedisCacheRepository_Set(t *testing.T) {
	ctx := context.Background()
	db, mock := redismock.NewClientMock()
	repo := NewRedisCacheRepository(db)

	key := "test:set"
	data := testStruct{Name: "Bob", Age: 25}
	jsonData, _ := json.Marshal(data) // 實際存入 Redis 的是 byte slice
	ttl := 5 * time.Minute

	// 測試案例 1: 成功 Set
	t.Run("Success", func(t *testing.T) {
		// ***  : 期望參數從 string(jsonData) 改為 jsonData ([]byte) ***
		mock.ExpectSet(key, jsonData, ttl).SetVal("OK") // <-- 直接使用 []byte

		err := repo.Set(ctx, key, data, ttl)

		assert.NoError(t, err)                        // <-- 檢查 Set 操作本身有無 mock 匹配錯誤等
		assert.NoError(t, mock.ExpectationsWereMet()) // <-- 檢查預期是否滿足
	})

	// 測試案例 2: JSON Marshal 錯誤 (保持不變)
	t.Run("Marshal Error", func(t *testing.T) {
		unmarshallableValue := make(chan int)

		err := repo.Set(ctx, key, unmarshallableValue, ttl)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis marshal failed")
	})

	// 測試案例 3: Redis 錯誤
	t.Run("Redis Error", func(t *testing.T) {
		expectedErr := errors.New("redis set error")
		// ***  : 期望參數從 string(jsonData) 改為 jsonData ([]byte) ***
		mock.ExpectSet(key, jsonData, ttl).SetErr(expectedErr) // <-- 直接使用 []byte

		err := repo.Set(ctx, key, data, ttl)

		assert.Error(t, err)                                // 斷言有錯誤發生
		assert.Contains(t, err.Error(), "redis set failed") // 檢查錯誤訊息是否被包裝
		assert.ErrorIs(t, err, expectedErr)                 // <-- 檢查是否包含預設的 Redis 錯誤
		assert.NoError(t, mock.ExpectationsWereMet())       // <-- 檢查預期是否滿足 (即使出錯也要滿足)
	})
}

// 測試 Delete 方法
func TestRedisCacheRepository_Delete(t *testing.T) {
	ctx := context.Background()
	db, mock := redismock.NewClientMock()
	repo := NewRedisCacheRepository(db)

	keys := []string{"test:del:1", "test:del:2"}

	// 測試案例 1: 成功刪除多個 Keys
	t.Run("Delete Multiple Keys Success", func(t *testing.T) {
		mock.ExpectDel(keys...).SetVal(int64(len(keys))) // <-- 設置 Del 的預期行為

		err := repo.Delete(ctx, keys...)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// 測試案例 2: 成功刪除單個 Key
	t.Run("Delete Single Key Success", func(t *testing.T) {
		key := "test:del:single"
		mock.ExpectDel(key).SetVal(1) // <-- 設置 Del 的預期行為

		err := repo.Delete(ctx, key)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// 測試案例 3: 刪除不存在的 Key (Redis 命令本身是成功的)
	t.Run("Delete Non-Existent Key Success", func(t *testing.T) {
		key := "test:del:nonexistent"
		mock.ExpectDel(key).SetVal(0) // Redis 返回 0 表示沒有 key 被刪除

		err := repo.Delete(ctx, key)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// 測試案例 4: 不傳入 Key 刪除 (應直接返回 nil)
	t.Run("Delete No Keys", func(t *testing.T) {
		err := repo.Delete(ctx) // 不傳入 keys

		assert.NoError(t, err)
		// 沒有 Redis 操作，不需要 mock 預期
	})

	// 測試案例 5: Redis 錯誤
	t.Run("Redis Error", func(t *testing.T) {
		expectedErr := errors.New("redis del error")
		mock.ExpectDel(keys...).SetErr(expectedErr) // <-- 設置 Del 返回錯誤

		err := repo.Delete(ctx, keys...)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis delete failed")
		assert.ErrorIs(t, err, expectedErr)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
