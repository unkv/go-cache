package cache

import (
	"sync"
	"time"
)

// ExpiredCallback Callback the function when the key-value pair expires
// Note that it is executed after expiration
type ExpiredCallback[K comparable, V any] func(k K, v V) error

type memCacheShard[K comparable, V any] struct {
	hashmap         map[K]Item[V]
	lock            sync.RWMutex
	expiredCallback ExpiredCallback[K, V]
}

func newMemCacheShard[K comparable, V any](conf *Config[K, V]) *memCacheShard[K, V] {
	return &memCacheShard[K, V]{expiredCallback: conf.expiredCallback, hashmap: map[K]Item[V]{}}
}

func (c *memCacheShard[K, V]) set(k K, item *Item[V]) {
	c.lock.Lock()
	c.hashmap[k] = *item
	c.lock.Unlock()
	return
}

func getZero[T any]() T {
	var result T
	return result
}

func (c *memCacheShard[K, V]) get(k K) (V, bool) {
	c.lock.RLock()
	item, exist := c.hashmap[k]
	c.lock.RUnlock()
	if !exist {
		return getZero[V](), false
	}
	if !item.Expired() {
		return item.v, true
	}
	if c.delExpired(k) {
		return getZero[V](), false
	}
	return c.get(k)
}

// func (c *memCacheShard[K, V]) getSet(k K, item *Item[V]) (V, bool) {
// 	defer c.set(k, item)
// 	return c.get(k)
// }

// func (c *memCacheShard[K, V]) getDel(k K) (V, bool) {
// 	defer c.del(k)
// 	return c.get(k)
// }

func (c *memCacheShard[K, V]) del(k K) int {
	var count int
	c.lock.Lock()
	v, found := c.hashmap[k]
	if found {
		delete(c.hashmap, k)
		if !v.Expired() {
			count++
		}
	}
	c.lock.Unlock()
	return count
}

// delExpired Only delete when key expires
func (c *memCacheShard[K, V]) delExpired(k K) bool {
	c.lock.Lock()
	item, found := c.hashmap[k]
	if !found || !item.Expired() {
		c.lock.Unlock()
		return false
	}
	delete(c.hashmap, k)
	c.lock.Unlock()
	if c.expiredCallback != nil {
		_ = c.expiredCallback(k, item.v)
	}
	return true
}

func (c *memCacheShard[K, V]) ttl(k K) (time.Duration, bool) {
	c.lock.RLock()
	v, found := c.hashmap[k]
	c.lock.RUnlock()
	if !found || !v.CanExpire() || v.Expired() {
		return 0, false
	}
	return time.Until(v.expire), true
}

func (c *memCacheShard[K, V]) checkExpire() {
	var expiredKeys []K
	c.lock.RLock()
	for k, item := range c.hashmap {
		if item.Expired() {
			expiredKeys = append(expiredKeys, k)
		}
	}
	c.lock.RUnlock()
	for _, k := range expiredKeys {
		c.delExpired(k)
	}
}

func (c *memCacheShard[K, V]) saveToMap(target map[K]V) {
	c.lock.RLock()
	for k, item := range c.hashmap {
		if item.Expired() {
			continue
		}
		target[k] = item.v
	}
	c.lock.RUnlock()
}
