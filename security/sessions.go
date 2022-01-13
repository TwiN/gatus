package security

import "github.com/TwiN/gocache/v2"

var sessions = gocache.NewCache().WithEvictionPolicy(gocache.LeastRecentlyUsed) // TODO: Move this to storage
