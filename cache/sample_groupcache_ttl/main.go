package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang/groupcache"
	"github.com/patrickmn/go-cache"
)

// 查詢數據的函數
func fetchData(key string) string {
	// 模擬數據查詢
	fmt.Println("query data...")
	time.Sleep(1 * time.Second) // 模擬查詢延遲
	return "Data for key: " + key
}

// 創建 Groupcache GetterFunc
func createGetterFunc(externalCache *cache.Cache) groupcache.GetterFunc {
	return groupcache.GetterFunc(func(ctx context.Context, key string, dest groupcache.Sink) error {
		// 如果外部緩存中有數據，直接返回
		if val, found := externalCache.Get(key); found {
			dest.SetString(val.(string))
			return nil
		}

		// 查詢數據並存入外部緩存
		data := fetchData(key)
		externalCache.Set(key, data, cache.DefaultExpiration)
		dest.SetString(data)
		return nil
	})
}

func main() {
	// 創建外部緩存（過期 5 秒，檢查間隔 10 分鐘）
	externalCache := cache.New(5*time.Second, 10*time.Minute)

	// 創建 groupcache 緩存，並使用抽取的 GetterFunc
	// cacheGroup := groupcache.NewGroup("myCacheGroup", 64<<20, createGetterFunc(externalCache))

	// 將 groupcache.NewGroup 的緩存大小設為 0，以禁用內部緩存
	cacheGroup := groupcache.NewGroup("myCacheGroup", 0, createGetterFunc(externalCache))

	// 測試請求數據
	var data string
	ctx := context.Background()
	key := "exampleKey"

	// 第一次請求
	err := cacheGroup.Get(ctx, key, groupcache.StringSink(&data))
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	fmt.Println("Got from cache:", data)

	// 第二次請求: 從 cache 拿
	err = cacheGroup.Get(ctx, key, groupcache.StringSink(&data))
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	fmt.Println("Got from cache after expiration:", data)

	// 等待緩存過期
	time.Sleep(6 * time.Second)

	// 再次請求數據（會觸發 GetterFunc）
	err = cacheGroup.Get(ctx, key, groupcache.StringSink(&data))
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	fmt.Println("Got from cache after expiration:", data)
}
