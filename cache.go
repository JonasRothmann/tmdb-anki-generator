package tmdbankigenerator

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Cache[K comparable, T any] struct {
	content  map[K]T
	fileName string
	mu       sync.Mutex

	fileLock sync.Mutex
}

func NewCache[K comparable, T any](fileName string) (*Cache[K, T], error) {
	bytes, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var content map[K]T
	if err := json.Unmarshal(bytes, &content); err != nil {
		return nil, err
	}

	return &Cache[K, T]{
		content:  content,
		fileName: fileName,
	}, nil
}

func (c *Cache[K, T]) Get(key K) (bool, T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if value, ok := c.content[key]; ok == true {
		return true, value
	} else {
		return false, value
	}
}

func (c *Cache[K, T]) Set(key K, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.content[key] = value

	go func() {
		c.fileLock.Lock()
		defer c.fileLock.Unlock()

		fi, err := os.OpenFile(c.fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			fmt.Printf("cache failed to open file: %s", err)
			//return err
		}
		defer fi.Close()

		jsonContent, err := json.Marshal(c.content)
		if err != nil {
			fmt.Printf("cache failed to marshal json content for saving to disk: %s", err)
		}

		if _, err := fi.WriteString(string(jsonContent)); err != nil {
			fmt.Printf("cache failed to write to file: %s", err)
			//return err
		}
	}()
}
