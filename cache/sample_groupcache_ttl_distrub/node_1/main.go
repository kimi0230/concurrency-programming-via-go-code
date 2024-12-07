package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang/groupcache"
	"github.com/patrickmn/go-cache"
)

// 查詢數據的函數
func fetchData(key string) string {
	fmt.Println("query data from external source...")
	time.Sleep(1 * time.Second) // 模擬查詢延遲
	return "Data for key: " + key
}

// 創建 Groupcache GetterFunc
func createGetterFunc(localCache *cache.Cache) groupcache.GetterFunc {
	return groupcache.GetterFunc(func(ctx context.Context, key string, dest groupcache.Sink) error {
		// 先檢查本地緩存
		if val, found := localCache.Get(key); found {
			fmt.Println("Found in local cache:", key)
			dest.SetString(val.(string))
			return nil
		}

		// 本地緩存未命中，查詢數據
		data := fetchData(key)
		localCache.Set(key, data, cache.DefaultExpiration)
		dest.SetString(data)
		return nil
	})
}

func main() {
	// 配置當前節點的地址（需根據不同節點修改）
	selfAddress := "http://localhost:8081" // 節點 1 的地址
	peerAddresses := []string{
		"http://localhost:8081", // 節點 1
		"http://localhost:8082", // 節點 2
	}

	// 創建本地緩存（go-cache）
	localCache := cache.New(50*time.Second, 10*time.Minute)

	// 配置 Groupcache HTTP Pool
	peerPool := groupcache.NewHTTPPool(selfAddress)
	peerPool.Set(peerAddresses...) // 註冊其他節點地址

	// 創建 Groupcache 緩存
	cacheGroup := groupcache.NewGroup("myCacheGroup", 0, createGetterFunc(localCache))

	// 啟動 HTTP 服務
	go func() {
		log.Println("Starting server at", selfAddress)
		log.Fatal(http.ListenAndServe(selfAddress[len("http://"):], peerPool))
	}()

	// 測試緩存請求
	var data string
	ctx := context.Background()
	key := "exampleKey"

	// 第一次請求
	err := cacheGroup.Get(ctx, key, groupcache.StringSink(&data))
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	fmt.Println("First request, got data:", data)

	// 第二次請求
	err = cacheGroup.Get(ctx, key, groupcache.StringSink(&data))
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	fmt.Println("Second request, got data:", data)

	// 等待本地緩存過期
	time.Sleep(6 * time.Second)

	// 第三次請求
	err = cacheGroup.Get(ctx, key, groupcache.StringSink(&data))
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	fmt.Println("Third request after local cache expired, got data:", data)
}
