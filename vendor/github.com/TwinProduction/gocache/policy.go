package gocache

type EvictionPolicy string

var (
	LeastRecentlyUsed EvictionPolicy = "LeastRecentlyUsed"
	FirstInFirstOut   EvictionPolicy = "FirstInFirstOut"
)
