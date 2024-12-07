package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	var p sync.Pool
	p.New = func() interface{} {
		return &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	var wg sync.WaitGroup
	wg.Add(10)

	go func() {
		for i := 0; i < 10; i++ {
			go func(i int) {
				defer wg.Done()
				c := p.Get().(*http.Client)
				defer p.Put(c)

				url := "http://bing.com"
				resp, err := c.Get(url)
				if err != nil {
					fmt.Printf("failed to get %s %s\n", url, err)
					return
				}
				resp.Body.Close()
				fmt.Printf("[%d] %s : %d \n", i, url, resp.StatusCode)
			}(i)
		}
	}()

	wg.Wait()
}
