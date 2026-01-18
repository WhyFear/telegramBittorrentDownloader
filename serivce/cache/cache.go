package cache

import "github.com/maypok86/otter/v2"

type Cache struct {
	OtterCache *otter.Cache[string, string]
}

type Cacher interface {
	Get(key string) (string, error)
	Set(key string, value string) error
	SetDual(key string, value string) error
}
