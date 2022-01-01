package security

import "github.com/TwiN/gocache/v2"

var sessions = gocache.NewCache() // TODO: Move this to storage
