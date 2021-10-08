package gocache

// EvictionPolicy is what dictates how evictions are handled
type EvictionPolicy string

var (
	// LeastRecentlyUsed is an eviction policy that causes the most recently accessed cache entry to be moved to the
	// head of the cache. Effectively, this causes the cache entries that have not been accessed for some time to
	// gradually move closer and closer to the tail, and since the tail is the entry that gets deleted when an eviction
	// is required, it allows less used cache entries to be evicted while keeping recently accessed entries at or close
	// to the head.
	//
	// For instance, creating a Cache with a Cache.MaxSize of 3 and creating the entries 1, 2 and 3 in that order would
	// put 3 at the head and 1 at the tail:
	//     3 (head) -> 2 -> 1 (tail)
	// If the cache entry 1 was then accessed, 1 would become the head and 2 the tail:
	//     1 (head) -> 3 -> 2 (tail)
	// If a cache entry 4 was then created, because the Cache.MaxSize is 3, the tail (2) would then be evicted:
	//     4 (head) -> 1 -> 3 (tail)
	LeastRecentlyUsed EvictionPolicy = "LeastRecentlyUsed"

	// FirstInFirstOut is an eviction policy that causes cache entries to be evicted in the same order that they are
	// created.
	//
	// For instance, creating a Cache with a Cache.MaxSize of 3 and creating the entries 1, 2 and 3 in that order would
	// put 3 at the head and 1 at the tail:
	//     3 (head) -> 2 -> 1 (tail)
	// If the cache entry 1 was then accessed, unlike with LeastRecentlyUsed, nothing would change:
	//     3 (head) -> 2 -> 1 (tail)
	// If a cache entry 4 was then created, because the Cache.MaxSize is 3, the tail (1) would then be evicted:
	//     4 (head) -> 3 -> 2 (tail)
	FirstInFirstOut EvictionPolicy = "FirstInFirstOut"
)
