package cache

import "time"

// SetIOption The option used to cache set
type SetIOption[K comparable, V any] func(ICache[K, V], K, IItem) bool

// WithEx Set the specified expire time, in time.Duration.
func WithEx[K comparable, V any](d time.Duration) SetIOption[K, V] {
	return func(c ICache[K, V], k K, v IItem) bool {
		v.SetExpireAt(time.Now().Add(d))
		return true
	}
}

// WithExAt Set the specified expire deadline, in time.Time.
func WithExAt[K comparable, V any](t time.Time) SetIOption[K, V] {
	return func(c ICache[K, V], k K, v IItem) bool {
		v.SetExpireAt(t)
		return true
	}
}

// ICacheOption The option used to create the cache object
type ICacheOption[K comparable, V any] func(conf *Config[K, V])

// WithShards set custom size of sharding. Default is 1024
// The larger the size, the smaller the lock force, the higher the concurrency performance,
// and the higher the memory footprint, so try to choose a size that fits your business scenario
func WithShards[K comparable, V any](shards int) ICacheOption[K, V] {
	if shards <= 0 {
		panic("Invalid shards")
	}
	return func(conf *Config[K, V]) {
		conf.shards = shards
	}
}

// WithExpiredCallback set custom expired callback function
// This callback function is called when the key-value pair expires
func WithExpiredCallback[K comparable, V any](ec ExpiredCallback[K, V]) ICacheOption[K, V] {
	return func(conf *Config[K, V]) {
		conf.expiredCallback = ec
	}
}

// WithHash set custom hash key function
func WithHash[K comparable, V any](hash IHash[K]) ICacheOption[K, V] {
	return func(conf *Config[K, V]) {
		conf.hash = hash
	}
}

// WithClearInterval set custom clear interval.
// Interval for clearing expired key-value pairs. The default value is 1 second
// If the d is 0, the periodic clearing function is disabled
func WithClearInterval[K comparable, V any](d time.Duration) ICacheOption[K, V] {
	return func(conf *Config[K, V]) {
		conf.clearInterval = d
	}
}
