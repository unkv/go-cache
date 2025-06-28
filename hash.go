package cache

import "fmt"

// IHash is responsible for generating unsigned, 64-bit hash of provided string. IHash should minimize collisions
// (generating same hash for different strings) and while performance is also important fast functions are preferable (i.e.
// you can use FarmHash family).
type IHash[K comparable] interface {
	Sum64(K) uint64
}

// DefaultStringHash returns a new 64-bit FNV-1a IHash which makes no memory allocations.
// Its Sum64 method will lay the value out in big-endian byte order.
// See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function
func DefaultStringHash() IHash[string] {
	return fnv64a{}
}

type fnv64a struct{}

const (
	// offset64 FNVa offset basis. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	offset64 = 14695981039346656037
	// prime64 FNVa prime value. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	prime64 = 1099511628211
)

// Sum64 gets the string and returns its uint64 hash value.
func (f fnv64a) Sum64(key string) uint64 {
	var hash uint64 = offset64
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}

	return hash
}

// toString is a IHash that converts the key to a string and then hashes it
type toString[K comparable] struct{}

func (toString[K]) Sum64(k K) uint64 {
	return DefaultStringHash().Sum64(fmt.Sprintf("%v", k))
}

// NewDefaultHash returns a new IHash that converts the key to a string and then hashes it
func NewDefaultHash[K comparable]() IHash[K] {
	return toString[K]{}
}
