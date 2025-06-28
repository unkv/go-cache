package cache

import (
	"runtime"
	"time"
)

type ICache[K comparable, V any] interface {
	//Set key to hold the string value. If key already holds a value, it is overwritten, regardless of its type.
	//Any previous time to live associated with the key is discarded on successful SET operation.
	//Example:
	//c.Set("demo", 1)
	//c.Set("demo", 1, WithEx(10*time.Second))
	//c.Set("demo", 1, WithEx(10*time.Second), WithNx())
	Set(k K, v V, opts ...SetIOption[K, V]) bool
	//Get the value of key.
	//If the key does not exist the special value nil,false is returned.
	//Example:
	//c.Get("demo") //nil, false
	//c.Set("demo", "value")
	//c.Get("demo") //"value", true
	Get(k K) (V, bool)
	//GetSet Atomically sets key to value and returns the old value stored at key.
	//Returns nil,false when key not exists.
	//Example:
	//c.GetSet("demo", 1) //nil,false
	//c.GetSet("demo", 2) //1,true
	GetSet(k K, v V, opts ...SetIOption[K, V]) (V, bool)
	//GetDel Get the value of key and delete the key.
	//This command is similar to GET, except for the fact that it also deletes the key on success.
	//Example:
	//c.Set("demo", "value")
	//c.GetDel("demo") //"value", true
	//c.GetDel("demo") //nil, false
	GetDel(k K) (V, bool)
	//Del Removes the specified keys. A key is ignored if it does not exist.
	//Return the number of keys that were removed.
	//Example:
	//c.Set("demo1", "1")
	//c.Set("demo2", "1")
	//c.Del("demo1", "demo2", "demo3") //2
	Del(keys ...K) int
	//DelExpired Only delete when key expires
	//Example:
	//c.Set("demo1", "1")
	//c.Set("demo2", "1", WithEx(1*time.Second))
	//time.Sleep(1*time.Second)
	//c.DelExpired("demo1", "demo2") //true
	DelExpired(k K) bool
	//Exists Returns if key exists.
	//Return the number of exists keys.
	//Example:
	//c.Set("demo1", "1")
	//c.Set("demo2", "1")
	//c.Exists("demo1", "demo2", "demo3") //2
	Exists(keys ...K) bool
	//Expire Set a timeout on key.
	//After the timeout has expired, the key will automatically be deleted.
	//Return false if the key not exist.
	//Example:
	//c.Expire("demo", 1*time.Second) // false
	//c.Set("demo", "1")
	//c.Expire("demo", 1*time.Second) // true
	Expire(k K, d time.Duration) bool
	//ExpireAt has the same effect and semantic as Expire, but instead of specifying the number of seconds representing the TTL (time to live),
	//it takes an absolute Unix Time (seconds since January 1, 1970). A Time in the past will delete the key immediately.
	//Return false if the key not exist.
	//Example:
	//c.ExpireAt("demo", time.Now().Add(10*time.Second)) // false
	//c.Set("demo", "1")
	//c.ExpireAt("demo", time.Now().Add(10*time.Second)) // true
	ExpireAt(k K, t time.Time) bool
	//Persist Remove the existing timeout on key.
	//Return false if the key not exist.
	//Example:
	//c.Persist("demo") // false
	//c.Set("demo", "1")
	//c.Persist("demo") // true
	Persist(k K) bool
	//Ttl Returns the remaining time to live of a key that has a timeout.
	//Returns 0,false if the key does not exist or if the key exist but has no associated expire.
	//Example:
	//c.Set("demo", "1")
	//c.Ttl("demo") // 0,false
	//c.Set("demo", "1", WithEx(10*time.Second))
	//c.Ttl("demo") // 10*time.Second,true
	Ttl(k K) (time.Duration, bool)
	// ToMap converts the current cache into a map with string keys and interface{} values.
	// Where the keys are the field names (as strings) and the values are the corresponding data.
	// Returns:
	//   A map[string]interface{} where each entry corresponds to a field of the object and its value.
	// Example:
	// c.Set("a", 1)
	// c.Set("b", "uu")
	// c.ToMap() return {"a":1,"b":"uu"}
	// c.Expire("a", 1*time.Second)
	// when sleep(1*time.Second)
	// c.ToMap() return {"b":"uu"}
	ToMap() map[K]V
}

func NewMemCache[K comparable, V any](opts ...ICacheOption[K, V]) ICache[K, V] {
	conf := NewConfig[K, V]()
	for _, opt := range opts {
		opt(conf)
	}

	c := &memCache[K, V]{
		shards:    make([]*memCacheShard[K, V], conf.shards),
		closed:    make(chan struct{}),
		shardMask: uint64(conf.shards - 1),
		config:    conf,
		hash:      conf.hash,
	}
	for i := 0; i < len(c.shards); i++ {
		c.shards[i] = newMemCacheShard(conf)
	}
	if conf.clearInterval > 0 {
		go func() {
			ticker := time.NewTicker(conf.clearInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					for _, shard := range c.shards {
						shard.checkExpire()
					}
				case <-c.closed:
					return
				}
			}
		}()
	}
	cache := &MemCache[K, V]{c}
	// Associated finalizer function with obj.
	// When the obj is unreachable, close the obj.
	runtime.SetFinalizer(cache, func(cache *MemCache[K, V]) { close(cache.closed) })
	return cache
}

func NewWithStrKey[V any](opts ...ICacheOption[string, V]) ICache[string, V] {
	allOpts := make([]ICacheOption[string, V], 0, len(opts)+1)
	allOpts = append(allOpts, WithHash[string, V](DefaultStringHash()))
	allOpts = append(allOpts, opts...)
	return NewMemCache(allOpts...)
}

type MemCache[K comparable, V any] struct {
	*memCache[K, V]
}

type memCache[K comparable, V any] struct {
	shards    []*memCacheShard[K, V]
	hash      IHash[K]
	shardMask uint64
	config    *Config[K, V]
	closed    chan struct{}
}

func (c *memCache[K, V]) Set(k K, v V, opts ...SetIOption[K, V]) bool {
	item := Item[V]{v: v}
	for _, opt := range opts {
		if pass := opt(c, k, &item); !pass {
			return false
		}
	}
	hashedKey := c.hash.Sum64(k)
	shard := c.getShard(hashedKey)
	shard.set(k, &item)
	return true
}

func (c *memCache[K, V]) Get(k K) (V, bool) {
	hashedKey := c.hash.Sum64(k)
	shard := c.getShard(hashedKey)
	return shard.get(k)
}

func (c *memCache[K, V]) GetSet(k K, v V, opts ...SetIOption[K, V]) (V, bool) {
	defer c.Set(k, v, opts...)
	return c.Get(k)
}

func (c *memCache[K, V]) GetDel(k K) (V, bool) {
	defer c.Del(k)
	return c.Get(k)
}

func (c *memCache[K, V]) Del(ks ...K) int {
	var count int
	for _, k := range ks {
		hashedKey := c.hash.Sum64(k)
		shard := c.getShard(hashedKey)
		count += shard.del(k)
	}
	return count
}

// DelExpired Only delete when key expires
func (c *memCache[K, V]) DelExpired(k K) bool {
	hashedKey := c.hash.Sum64(k)
	shard := c.getShard(hashedKey)
	return shard.delExpired(k)
}

func (c *memCache[K, V]) Exists(ks ...K) bool {
	for _, k := range ks {
		if _, found := c.Get(k); !found {
			return false
		}
	}
	return true
}

func (c *memCache[K, V]) Expire(k K, d time.Duration) bool {
	v, found := c.Get(k)
	if !found {
		return false
	}
	return c.Set(k, v, WithEx[K, V](d))
}

func (c *memCache[K, V]) ExpireAt(k K, t time.Time) bool {
	v, found := c.Get(k)
	if !found {
		return false
	}
	return c.Set(k, v, WithExAt[K, V](t))
}

func (c *memCache[K, V]) Persist(k K) bool {
	v, found := c.Get(k)
	if !found {
		return false
	}
	return c.Set(k, v)
}

func (c *memCache[K, V]) Ttl(k K) (time.Duration, bool) {
	hashedKey := c.hash.Sum64(k)
	shard := c.getShard(hashedKey)
	return shard.ttl(k)
}

func (c *memCache[K, V]) ToMap() map[K]V {
	result := make(map[K]V)
	for _, shard := range c.shards {
		shard.saveToMap(result)
	}
	return result
}

func (c *memCache[K, V]) getShard(hashedKey uint64) (shard *memCacheShard[K, V]) {
	return c.shards[hashedKey&c.shardMask]
}
