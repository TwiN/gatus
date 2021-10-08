package gocache

type Statistics struct {
	// EvictedKeys is the number of keys that were evicted
	EvictedKeys uint64

	// ExpiredKeys is the number of keys that were automatically deleted as a result of expiring
	ExpiredKeys uint64

	// Hits is the number of cache hits
	Hits uint64

	// Misses is the number of cache misses
	Misses uint64
}
