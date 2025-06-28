package cache

import "time"

type Config[K comparable, V any] struct {
	shards          int
	expiredCallback ExpiredCallback[K, V]
	hash            IHash[K]
	clearInterval   time.Duration
}

func NewConfig[K comparable, V any]() *Config[K, V] {
	return &Config[K, V]{shards: 1024, hash: NewDefaultHash[K](), clearInterval: 1 * time.Second}
}
