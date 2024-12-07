package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang/groupcache"
	"github.com/patrickmn/go-cache"
)

var (
	peerPool         *groupcache.HTTPPool // 全局的 HTTP Pool
	once             sync.Once            // 確保 HTTP Pool 只初始化一次
	forceDeleteKey   string               // 強制刪除的鍵
	existingGroupMap = sync.Map{}         // 跟蹤已創建的組，避免重複
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
		// 如果 forceDeleteKey 被設置，跳過緩存
		if forceDeleteKey == "*" || key == forceDeleteKey {
			fmt.Println("Forcing deletion for key:", key)
			localCache.Delete(key)
			forceDeleteKey = "" // 重置標誌
			return fmt.Errorf("Key %s was deleted", key)
		}

		// 檢查本地緩存
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

// 動態生成唯一組名
func generateUniqueGroupName(baseName string) string {
	return fmt.Sprintf("%s_%d", baseName, time.Now().UnixNano())
}

// 初始化緩存
func initializeCache(selfAddress string, peerAddresses []string, localCache *cache.Cache, maxCacheSize int64, groupName string) *groupcache.Group {
	// 確保 HTTPPool 只初始化一次
	once.Do(func() {
		peerPool = groupcache.NewHTTPPool(selfAddress)
		// 更新節點配置
		peerPool.Set(peerAddresses...)
	})

	// 檢查組是否已存在
	if group := groupcache.GetGroup(groupName); group != nil {
		fmt.Println("Group already exists:", groupName)
		return group
	}

	// 創建新的 Group
	fmt.Println("Creating new group:", groupName)
	group := groupcache.NewGroup(groupName, maxCacheSize, createGetterFunc(localCache))
	existingGroupMap.Store(groupName, group)
	return group
}

// 重置分散式緩存
func resetGroupCache(selfAddress string, peerAddresses []string, localCache *cache.Cache, maxCacheSize int64, groupName string) *groupcache.Group {
	fmt.Println("Resetting groupcache...")
	existingGroupMap.Delete(groupName) // 刪除記錄
	return initializeCache(selfAddress, peerAddresses, localCache, maxCacheSize, groupName)
}

// 清空本地緩存
func clearLocalCache(localCache *cache.Cache) {
	localCache.Flush()
	fmt.Println("Local cache cleared")
}

func main() {
	// 配置節點地址
	selfAddress := "http://localhost:8081"
	peerAddresses := []string{"http://localhost:8081", "http://localhost:8082"}
	groupName := "myCacheGroup" // 組名稱

	// 創建本地緩存（go-cache）
	localCache := cache.New(50*time.Second, 10*time.Minute)

	// 初始化分散式緩存
	cacheGroup := initializeCache(selfAddress, peerAddresses, localCache, 0, groupName)

	// 啟動 HTTP 服務
	go func() {
		log.Println("Starting server at", selfAddress)
		log.Fatal(http.ListenAndServe(selfAddress[len("http://"):], peerPool))
	}()

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
	fmt.Println("Second request after local cache cleared, got data:", data)

	// 清空本地緩存
	clearLocalCache(localCache)
	// 重置分散式緩存
	cacheGroup = resetGroupCache(selfAddress, peerAddresses, localCache, 0, groupName)
	fmt.Println("Distributed cache reset completed.")

	time.Sleep(6 * time.Second) // 等待一段時間，確保分散式緩存已重置

	// 第三次請求
	err = cacheGroup.Get(ctx, key, groupcache.StringSink(&data))
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	fmt.Println("Third request after distributed cache reset, got data:", data)
}
