package cache

import (
	"fmt"
	"time"

	"github.com/maypok86/otter/v2"
)

func NewOtterCache() *Cache {
	cache := otter.Must(&otter.Options[string, string]{
		MaximumSize:      1_000, // 最大容量 1 万个对象
		InitialCapacity:  100,   // 初始分配空间
		ExpiryCalculator: otter.ExpiryAccessing[string, string](time.Hour),
	})
	return &Cache{
		OtterCache: cache,
	}
}

func (c *Cache) Get(key string) (string, error) {
	if val, ok := c.OtterCache.GetIfPresent(key); ok {
		return val, nil
	}
	return "", nil
}

func (c *Cache) Set(key string, value string) error {
	_, ok := c.OtterCache.Set(key, value)
	if !ok {
		return fmt.Errorf("set cache failed, key: %s, value: %s", key, value)
	}
	return nil
}

func (c *Cache) SetDual(key string, value string) error {
	err := c.Set(key, value)
	if err != nil {
		return err
	}
	err = c.Set(value, key)
	if err != nil {
		c.OtterCache.Invalidate(key)
		return err
	}
	return nil
}
