# gocache

![build](https://github.com/TwinProduction/gocache/workflows/build/badge.svg?branch=master) 
[![Go Report Card](https://goreportcard.com/badge/github.com/TwinProduction/gocache)](https://goreportcard.com/report/github.com/TwinProduction/gocache)
[![codecov](https://codecov.io/gh/TwinProduction/gocache/branch/master/graph/badge.svg)](https://codecov.io/gh/TwinProduction/gocache)
[![Go version](https://img.shields.io/github/go-mod/go-version/TwinProduction/gocache.svg)](https://github.com/TwinProduction/gocache)
[![Go Reference](https://pkg.go.dev/badge/github.com/TwinProduction/gocache.svg)](https://pkg.go.dev/github.com/TwinProduction/gocache)

gocache is an easy-to-use, high-performance, lightweight and thread-safe (goroutine-safe) in-memory key-value cache 
with support for LRU and FIFO eviction policies as well as expiration, bulk operations and even persistence to file.


## Table of Contents

- [Features](#features)
- [Usage](#usage)
  - [Initializing the cache](#initializing-the-cache)
  - [Functions](#functions)
  - [Examples](#examples)
    - [Creating or updating an entry](#creating-or-updating-an-entry)
    - [Getting an entry](#getting-an-entry)
    - [Deleting an entry](#deleting-an-entry)
    - [Complex example](#complex-example)
- [Persistence](#persistence)
  - [Limitations](#limitations)
- [Eviction](#eviction)
  - [MaxSize](#maxsize)
  - [MaxMemoryUsage](#maxmemoryusage)
- [Expiration](#expiration)
- [Server](#server)
- [Running the server with Docker](#running-the-server-with-docker)
- [Performance](#performance)
  - [Summary](#summary)
  - [Results](#results)
- [FAQ](#faq)
  - [Why does the memory usage not go down?](#why-does-the-memory-usage-not-go-down)


## Features
gocache supports the following cache eviction policies: 
- First in first out (FIFO)
- Least recently used (LRU)

It also supports cache entry TTL, which is both active and passive. Active expiration means that if you attempt 
to retrieve a cache key that has already expired, it will delete it on the spot and the behavior will be as if
the cache key didn't exist. As for passive expiration, there's a background task that will take care of deleting
expired keys.

It also includes what you'd expect from a cache, like bulk operations, persistence and patterns.

While meant to be used as a library, there's a Redis-compatible cache server included. 
See the [Server](#server) section. 
It may also serve as a good reference to use in order to implement gocache in your own applications.


## Usage
```
go get -u github.com/TwinProduction/gocache
```

### Initializing the cache
```go
cache := gocache.NewCache().WithMaxSize(1000).WithEvictionPolicy(gocache.LeastRecentlyUsed)
```

If you're planning on using expiration (`SetWithTTL` or `Expire`) and you want expired entries to be automatically deleted 
in the background, make sure to start the janitor when you instantiate the cache:

```go
cache.StartJanitor()
```

### Functions
| Function                          | Description |
| --------------------------------- | ----------- |
| WithMaxSize                       | Sets the max size of the cache. `gocache.NoMaxSize` means there is no limit. If not set, the default max size is `gocache.DefaultMaxSize`.
| WithMaxMemoryUsage                | Sets the max memory usage of the cache. `gocache.NoMaxMemoryUsage` means there is no limit. The default behavior is to not evict based on memory usage.
| WithEvictionPolicy                | Sets the eviction algorithm to be used when the cache reaches the max size. If not set, the default eviction policy is `gocache.FirstInFirstOut` (FIFO).
| WithForceNilInterfaceOnNilPointer | Configures whether values with a nil pointer passed to write functions should be forcefully set to nil. Defaults to true.
| StartJanitor                      | Starts the janitor, which is in charge of deleting expired cache entries in the background.
| StopJanitor                       | Stops the janitor.
| Set                               | Same as `SetWithTTL`, but with no expiration (`gocache.NoExpiration`)
| SetAll                            | Same as `Set`, but in bulk
| SetWithTTL                        | Creates or updates a cache entry with the given key, value and expiration time. If the max size after the aforementioned operation is above the configured max size, the tail will be evicted. Depending on the eviction policy, the tail is defined as the oldest 
| Get                               | Gets a cache entry by its key.
| GetByKeys                         | Gets a map of entries by their keys. The resulting map will contain all keys, even if some of the keys in the slice passed as parameter were not present in the cache.  
| GetAll                            | Gets all cache entries.
| GetKeysByPattern                  | Retrieves a slice of keys that matches a given pattern.
| Delete                            | Removes a key from the cache.
| DeleteAll                         | Removes multiple keys from the cache.
| Count                             | Gets the size of the cache. This includes cache keys which may have already expired, but have not been removed yet.
| Clear                             | Wipes the cache.
| TTL                               | Gets the time until a cache key expires. 
| Expire                            | Sets the expiration time of an existing cache key.
| SaveToFile                        | Stores the content of the cache to a file so that it can be read using `ReadFromFile`. See [persistence](#persistence).
| ReadFromFile                      | Populates the cache using a file created using `SaveToFile`. See [persistence](#persistence).

For further documentation, please refer to [Go Reference](https://pkg.go.dev/github.com/TwinProduction/gocache)


### Examples

#### Creating or updating an entry
```go
cache.Set("key", "value") 
cache.Set("key", 1)
cache.Set("key", struct{ Text string }{Test: "value"})
cache.SetWithTTL("key", []byte("value"), 24*time.Hour)
```

#### Getting an entry
```go
value, exists := cache.Get("key")
```
You can also get multiple entries by using `cache.GetByKeys([]string{"key1", "key2"})`

#### Deleting an entry
```go
cache.Delete("key")
```
You can also delete multiple entries by using `cache.DeleteAll([]string{"key1", "key2"})`

#### Complex example
```go
package main

import (
	"fmt"
	"time"

    "github.com/TwinProduction/gocache"
)

func main() {
	cache := gocache.NewCache().WithEvictionPolicy(gocache.LeastRecentlyUsed).WithMaxSize(10000)
	cache.StartJanitor() // Passively manages expired entries

	cache.Set("key", "value")
	cache.SetWithTTL("key-with-ttl", "value", 60*time.Minute)
	cache.SetAll(map[string]interface{}{"k1": "v1", "k2": "v2", "k3": "v3"})

	value, exists := cache.Get("key")
	fmt.Printf("[Get] key=key; value=%s; exists=%v\n", value, exists)
	for key, value := range cache.GetByKeys([]string{"k1", "k2", "k3"}) {
		fmt.Printf("[GetByKeys] key=%s; value=%s\n", key, value)
	}
	for _, key := range cache.GetKeysByPattern("key*", 0) {
		fmt.Printf("[GetKeysByPattern] key=%s\n", key)
	}

	fmt.Println("Cache size before persisting cache to file:", cache.Count())
	err := cache.SaveToFile("cache.bak")
	if err != nil {
		panic(fmt.Sprintf("failed to persist cache to file: %s", err.Error()))
	}

	cache.Expire("key", time.Hour)
	time.Sleep(500*time.Millisecond)
	timeUntilExpiration, _ := cache.TTL("key")
	fmt.Println("Number of minutes before 'key' expires:", int(timeUntilExpiration.Seconds()))

	cache.Delete("key")
	cache.DeleteAll([]string{"k1", "k2", "k3"})

	fmt.Println("Cache size before restoring cache from file:", cache.Count())
	_, err = cache.ReadFromFile("cache.bak")
	if err != nil {
		panic(fmt.Sprintf("failed to restore cache from file: %s", err.Error()))
	}

	fmt.Println("Cache size after restoring cache from file:", cache.Count())
	cache.Clear()
	fmt.Println("Cache size after clearing the cache:", cache.Count())
}
```

<details>
  <summary>Output</summary>

```
[Get] key=key; value=value; exists=true
[GetByKeys] key=k2; value=v2
[GetByKeys] key=k3; value=v3
[GetByKeys] key=k1; value=v1
[GetKeysByPattern] key=key
[GetKeysByPattern] key=key-with-ttl
Cache size before persisting cache to file: 5
Number of minutes before 'key' expires: 3599
Cache size before restoring cache from file: 1
Cache size after restoring cache from file: 5
Cache size after clearing the cache: 0
```
</details>


## Persistence
While gocache is an in-memory cache, you can still save the content of the cache in a file
and vice versa.

To save the content of the cache to a file:
```go
err := cache.SaveToFile(TestCacheFile)
```

To retrieve the content of the cache from a file:
```go
numberOfEntriesEvicted, err := newCache.ReadFromFile(TestCacheFile)
```
The `numberOfEntriesEvicted` will be non-zero only if the number of entries 
in the file is higher than the cache's configured `MaxSize`.

### Limitations
While you can cache structs in memory out of the box, persisting structs to a file requires you to 
**register the custom interfaces that your application uses with the `gob` package**.

```go
type YourCustomStruct struct {
	A string
	B int
}

// ...
cache.Set("key", YourCustomStruct{A: "test", B: 123})
```
To persist your custom struct properly:
```go
gob.Register(YourCustomStruct{})
cache.SaveToFile("gocache.bak")
``` 
The same applies for restoring the cache from a file:
```go
cache := NewCache()
gob.Register(YourCustomStruct{})
cache.ReadFromFile(TestCacheFile)
value, _ := cache.Get("key")
fmt.Println(value.(YourCustomStruct))
```
You only need to persist the struct once, so adding the following function in a file would suffice:
```go
func init() {
    gob.Register(YourCustomStruct{})
}
```

Failure to register your custom structs will prevent gocache from persisting and/or parsing the value of each keys that 
use said custom structs.

That being said, assuming that you're using gocache as a cache, this shouldn't create any bugs on your end, because
every key that cannot be parsed are not populated into the cache by `ReadFromFile`.

In other words, if you're falling back to a database or something similar when the cache doesn't have the key requested,
you'll be fine.


## Eviction

### MaxSize
Eviction by MaxSize is the default behavior, and is also the most efficient.

The code below will create a cache that has a maximum size of 1000:
```go
cache := gocache.NewCache().WithMaxSize(1000)
```
This means that whenever an operation causes the total size of the cache to go above 1000, the tail will be evicted.

### MaxMemoryUsage
Eviction by MaxMemoryUsage is **disabled by default**, and is in alpha.

The code below will create a cache that has a maximum memory usage of 50MB:
```go
cache := gocache.NewCache().WithMaxSize(0).WithMaxMemoryUsage(50*gocache.Megabyte)
```
This means that whenever an operation causes the total memory usage of the cache to go above 50MB, one or more tails
will be evicted.

Unlike evictions caused by reaching the MaxSize, evictions triggered by MaxMemoryUsage may lead to multiple entries
being evicted in a row. The reason for this is that if, for instance, you had 100 entries of 0.1MB each and you suddenly added 
a single entry of 10MB, 100 entries would need to be evicted to make enough space for that new big entry.

It's very important to keep in mind that eviction by MaxMemoryUsage is approximate.

**The only memory taken into consideration is the size of the cache, not the size of the entire application.**
If you pass along 100MB worth of data in a matter of seconds, even though the cache's memory usage will remain
under 50MB (or whatever you configure the MaxMemoryUsage to), the memory footprint generated by that 100MB will 
still exist until the next GC cycle.

As previously mentioned, this is a work in progress, and here's a list of the things you should keep in mind:
- The memory usage of structs are a gross estimation and may not reflect the actual memory usage.
- Native types (string, int, bool, []byte, etc.) are the most accurate for calculating the memory usage.
- Adding an entry bigger than the configured MaxMemoryUsage will work, but it will evict all other entries.


## Expiration
There are two ways that the deletion of expired keys can take place:
- Active
- Passive

**Active deletion of expired keys** happens when an attempt is made to access the value of a cache entry that expired. 
`Get`, `GetByKeys` and `GetAll` are the only functions that can trigger active deletion of expired keys.

**Passive deletion of expired keys** runs in the background and is managed by the janitor. 
If you do not start the janitor, there will be no passive deletion of expired keys.


## Server
For the sake of convenience, a ready-to-go cache server is available 
through the `gocacheserver` package. 

The reason why the server is in a different package is because `gocache` does not use 
any external dependencies, but rather than re-inventing the wheel, the server 
implementation uses redcon, which is a Redis server framework for Go.

That way, those who desire to use gocache without the server will not add any extra dependencies
as long as they don't import the `gocacheserver` package. 

```go
package main

import (
	"github.com/TwinProduction/gocache"
	"github.com/TwinProduction/gocache/gocacheserver"
)

func main() {
	cache := gocache.NewCache().WithEvictionPolicy(gocache.LeastRecentlyUsed).WithMaxSize(100000)
	server := gocacheserver.NewServer(cache)
	server.Start()
}
```

Any Redis client should be able to interact with the server, though only the following instructions are supported:
- [X] GET
- [X] SET
- [X] DEL
- [X] PING
- [X] QUIT
- [X] INFO
- [X] EXPIRE
- [X] SETEX
- [X] TTL
- [X] FLUSHDB
- [X] EXISTS
- [X] ECHO
- [X] MGET
- [X] MSET
- [X] SCAN (kind of - cursor is not currently supported)
- [ ] KEYS


## Running the server with Docker
[![Docker pulls](https://img.shields.io/docker/pulls/twinproduction/gocache-server.svg)](https://cloud.docker.com/repository/docker/twinproduction/gocache-server)

To build it locally, refer to the Makefile's `docker-build` and `docker-run` steps.

Note that the server version of gocache is still under development.

```
docker run --name gocache-server -p 6379:6379 twinproduction/gocache-server
```


## Performance

### Summary
- **Set**: Both map and gocache have the same performance.
- **Get**: Map is faster than gocache.

This is because gocache keeps track of the head and the tail for eviction and expiration/TTL. 

Ultimately, the difference is negligible. 

We could add a way to disable eviction or disable expiration altogether just to match the map's performance, 
but if you're looking into using a library like gocache, odds are, you want more than just a map.


### Results
| key    | value    |
|:------ |:-------- |
| goos   | windows  |
| goarch | amd64    |
| cpu    | i7-9700K |
| mem    | 32G DDR4 |

```
BenchmarkMap_Get-8                                                         	95936680	      26.3 ns/op
BenchmarkMap_SetSmallValue-8                                               	 7738132	       424 ns/op
BenchmarkMap_SetMediumValue-8                                              	 7766346	       424 ns/op
BenchmarkMap_SetLargeValue-8                                               	 7947063	       435 ns/op
BenchmarkCache_Get-8                                                       	54549049	      45.7 ns/op
BenchmarkCache_SetSmallValue-8                                             	35225013	      69.2 ns/op
BenchmarkCache_SetMediumValue-8                                            	 5952064	       412 ns/op
BenchmarkCache_SetLargeValue-8                                             	 5969121	       411 ns/op
BenchmarkCache_GetUsingLRU-8                                               	54545949	      45.6 ns/op
BenchmarkCache_SetSmallValueUsingLRU-8                                     	 5909504	       419 ns/op
BenchmarkCache_SetMediumValueUsingLRU-8                                    	 5910885	       418 ns/op
BenchmarkCache_SetLargeValueUsingLRU-8                                     	 5867544	       419 ns/op
BenchmarkCache_SetSmallValueWhenUsingMaxMemoryUsage-8                      	 5477178	       462 ns/op
BenchmarkCache_SetMediumValueWhenUsingMaxMemoryUsage-8                     	 5417595	       475 ns/op
BenchmarkCache_SetLargeValueWhenUsingMaxMemoryUsage-8                      	 5215263	       479 ns/op
BenchmarkCache_SetSmallValueWithMaxSize10-8                                	10115574	       236 ns/op
BenchmarkCache_SetMediumValueWithMaxSize10-8                               	10242792	       241 ns/op
BenchmarkCache_SetLargeValueWithMaxSize10-8                                	10201894	       241 ns/op
BenchmarkCache_SetSmallValueWithMaxSize1000-8                              	 9637113	       253 ns/op
BenchmarkCache_SetMediumValueWithMaxSize1000-8                             	 9635175	       253 ns/op
BenchmarkCache_SetLargeValueWithMaxSize1000-8                              	 9598982	       260 ns/op
BenchmarkCache_SetSmallValueWithMaxSize100000-8                            	 7642584	       337 ns/op
BenchmarkCache_SetMediumValueWithMaxSize100000-8                           	 7407571	       344 ns/op
BenchmarkCache_SetLargeValueWithMaxSize100000-8                            	 7071360	       345 ns/op
BenchmarkCache_SetSmallValueWithMaxSize100000AndLRU-8                      	 7544194	       332 ns/op
BenchmarkCache_SetMediumValueWithMaxSize100000AndLRU-8                     	 7667004	       344 ns/op
BenchmarkCache_SetLargeValueWithMaxSize100000AndLRU-8                      	 7357642	       338 ns/op
BenchmarkCache_GetAndSetMultipleConcurrently-8                             	 1442306	      1684 ns/op
BenchmarkCache_GetAndSetConcurrentlyWithRandomKeysAndLRU-8                 	 5117271	       477 ns/op
BenchmarkCache_GetAndSetConcurrentlyWithRandomKeysAndFIFO-8                	 5228412	       475 ns/op
BenchmarkCache_GetAndSetConcurrentlyWithRandomKeysAndNoEvictionAndLRU-8    	 5139195	       529 ns/op
BenchmarkCache_GetAndSetConcurrentlyWithRandomKeysAndNoEvictionAndFIFO-8   	 5251639	       511 ns/op
BenchmarkCache_GetAndSetConcurrentlyWithFrequentEvictionsAndLRU-8          	 7384626	       334 ns/op
BenchmarkCache_GetAndSetConcurrentlyWithFrequentEvictionsAndFIFO-8         	 7361985	       332 ns/op
BenchmarkCache_GetConcurrentlyWithLRU-8                                    	 3370784	       726 ns/op
BenchmarkCache_GetConcurrentlyWithFIFO-8                                   	 3749994	       681 ns/op
BenchmarkCache_GetKeysThatDoNotExistConcurrently-8                         	17647344	       143 ns/op
```


## FAQ

### Why does the memory usage not go down?

> **NOTE**: As of Go 1.16, this will no longer apply. See [golang/go#42330](https://github.com/golang/go/issues/42330)

By default, Go uses `MADV_FREE` if the kernel supports it to release memory, which is significantly more efficient 
than using `MADV_DONTNEED`. Unfortunately, this means that RSS doesn't go down unless the OS actually needs the 
memory. 

Technically, the memory _is_ available to the kernel, even if it shows a high memory usage, but the OS will only
use that memory if it needs to. In the case that the OS does need the freed memory, the RSS will go down and you'll
notice the memory usage lowering.

[reference](https://github.com/golang/go/issues/33376#issuecomment-666455792)

You can reproduce this by following the steps below:
- Start gocacheserver
- Note the memory usage
- Create 500k keys
- Note the memory usage
- Flush the cache
- Note that the memory usage has not decreased, despite the cache being empty.

**Substituting gocache for a normal map will yield the same result.**

If the released memory still appearing as used is a problem for you, 
you can set the environment variable `GODEBUG` to `madvdontneed=1`.
