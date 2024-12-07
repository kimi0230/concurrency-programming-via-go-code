package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang/groupcache"
)

// 模擬資料庫查詢
func databaseQuery(key string) (string, error) {
	// 模擬查詢操作
	time.Sleep(2 * time.Second)
	return "Data for key: " + key, nil
}

func main() {
	// 創建一個緩存組
	cache := groupcache.NewGroup("myCacheGroup", 64<<20, // 64 MB 的緩存大小
		groupcache.GetterFunc(func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			// 如果緩存中沒有數據，則調用這個函數獲取數據
			data, err := databaseQuery(key)
			if err != nil {
				return err
			}
			dest.SetString(data) // 將結果存入緩存
			return nil
		}),
	)

	// 請求數據（Get 操作）
	var data string
	key := "exampleKey"
	ctx := context.Background()
	err := cache.Get(ctx, key, groupcache.StringSink(&data)) // 將結果存入 `data`
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	fmt.Println("Got from cache:", data)

	err = cache.Get(ctx, key, groupcache.StringSink(&data)) // 將結果存入 `data`
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}

	fmt.Println("Got from cache:", data)
}
