# gocache
![test](https://github.com/TwiN/gocache/workflows/test/badge.svg?branch=master) 
[![Go Report Card](https://goreportcard.com/badge/github.com/TwiN/gocache)](https://goreportcard.com/report/github.com/TwiN/gocache)
[![codecov](https://codecov.io/gh/TwiN/gocache/branch/master/graph/badge.svg)](https://codecov.io/gh/TwiN/gocache)
[![Go version](https://img.shields.io/github/go-mod/go-version/TwiN/gocache.svg)](https://github.com/TwiN/gocache)
[![Go Reference](https://pkg.go.dev/badge/github.com/TwiN/gocache.svg)](https://pkg.go.dev/github.com/TwiN/gocache/v2)
[![Follow TwiN](https://img.shields.io/github/followers/TwiN?label=Follow&style=social)](https://github.com/TwiN)

gocache is an easy-to-use, high-performance, lightweight and thread-safe (goroutine-safe) in-memory key-value cache 
with support for LRU and FIFO eviction policies as well as expiration, bulk operations and even retrieval of keys by pattern.


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
- [Eviction](#eviction)
  - [MaxSize](#maxsize)
  - [MaxMemoryUsage](#maxmemoryusage)
- [Expiration](#expiration)
- [Performance](#performance)
  - [Summary](#summary)
  - [Results](#results)
- [FAQ](#faq)
  - [How can I persist the data on application termination?](#how-can-i-persist-the-data-on-application-termination)


## Features
gocache supports the following cache eviction policies: 
- First in first out (FIFO)
- Least recently used (LRU)

It also supports cache entry TTL, which is both active and passive. Active expiration means that if you attempt 
to retrieve a cache key that has already expired, it will delete it on the spot and the behavior will be as if
the cache key didn't exist. As for passive expiration, there's a background task that will take care of deleting
expired keys.

It also includes what you'd expect from a cache, like GET/SET, bulk operations and get by pattern.


## Usage
```
go get -u github.com/TwiN/gocache/v2
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
| Function                          | Description                                                                                                                                                                                                                                                        |
|-----------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| WithMaxSize                       | Sets the max size of the cache. `gocache.NoMaxSize` means there is no limit. If not set, the default max size is `gocache.DefaultMaxSize`.                                                                                                                         |
| WithMaxMemoryUsage                | Sets the max memory usage of the cache. `gocache.NoMaxMemoryUsage` means there is no limit. The default behavior is to not evict based on memory usage.                                                                                                            |
| WithEvictionPolicy                | Sets the eviction algorithm to be used when the cache reaches the max size. If not set, the default eviction policy is `gocache.FirstInFirstOut` (FIFO).                                                                                                           |
| WithDefaultTTL                    | Sets the default TTL for each entry.                                                                                                                                                                                                                               |
| WithForceNilInterfaceOnNilPointer | Configures whether values with a nil pointer passed to write functions should be forcefully set to nil. Defaults to true.                                                                                                                                          |
| StartJanitor                      | Starts the janitor, which is in charge of deleting expired cache entries in the background.                                                                                                                                                                        |
| StopJanitor                       | Stops the janitor.                                                                                                                                                                                                                                                 |
| Set                               | Same as `SetWithTTL`, but using the default TTL (which is `gocache.NoExpiration`, unless configured otherwise).                                                                                                                                                    |
| SetWithTTL                        | Creates or updates a cache entry with the given key, value and expiration time. If the max size after the aforementioned operation is above the configured max size, the tail will be evicted. Depending on the eviction policy, the tail is defined as the oldest |
| SetAll                            | Same as `Set`, but in bulk.                                                                                                                                                                                                                                        |
| SetAllWithTTL                     | Same as `SetWithTTL`, but in bulk.                                                                                                                                                                                                                                 |
| Get                               | Gets a cache entry by its key.                                                                                                                                                                                                                                     |
| GetByKeys                         | Gets a map of entries by their keys. The resulting map will contain all keys, even if some of the keys in the slice passed as parameter were not present in the cache.                                                                                             |
| GetAll                            | Gets all cache entries.                                                                                                                                                                                                                                            |
| GetKeysByPattern                  | Retrieves a slice of keys that matches a given pattern.                                                                                                                                                                                                            |
| Delete                            | Removes a key from the cache.                                                                                                                                                                                                                                      |
| DeleteAll                         | Removes multiple keys from the cache.                                                                                                                                                                                                                              |
| DeleteKeysByPattern               | Removes all keys that that matches a given pattern.                                                                                                                                                                                                                |
| Count                             | Gets the size of the cache. This includes cache keys which may have already expired, but have not been removed yet.                                                                                                                                                |
| Clear                             | Wipes the cache.                                                                                                                                                                                                                                                   |
| TTL                               | Gets the time until a cache key expires.                                                                                                                                                                                                                           |
| Expire                            | Sets the expiration time of an existing cache key.                                                                                                                                                                                                                 |

For further documentation, please refer to [Go Reference](https://pkg.go.dev/github.com/TwiN/gocache)


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

    "github.com/TwiN/gocache/v2"
)

func main() {
    cache := gocache.NewCache().WithEvictionPolicy(gocache.LeastRecentlyUsed).WithMaxSize(10000)
    cache.StartJanitor() // Passively manages expired entries
    defer cache.StopJanitor()

    cache.Set("key", "value")
    cache.SetWithTTL("key-with-ttl", "value", 60*time.Minute)
    cache.SetAll(map[string]any{"k1": "v1", "k2": "v2", "k3": "v3"})

    fmt.Println("[Count] Cache size:", cache.Count())

    value, exists := cache.Get("key")
    fmt.Printf("[Get] key=key; value=%s; exists=%v\n", value, exists)
    for key, value := range cache.GetByKeys([]string{"k1", "k2", "k3"}) {
        fmt.Printf("[GetByKeys] key=%s; value=%s\n", key, value)
    }
    for _, key := range cache.GetKeysByPattern("key*", 0) {
        fmt.Printf("[GetKeysByPattern] pattern=key*; key=%s\n", key)
    }

    cache.Expire("key", time.Hour)
    time.Sleep(500*time.Millisecond)
    timeUntilExpiration, _ := cache.TTL("key")
    fmt.Println("[TTL] Number of minutes before 'key' expires:", int(timeUntilExpiration.Seconds()))

    cache.Delete("key")
    cache.DeleteAll([]string{"k1", "k2", "k3"})
    
    cache.Clear()
    fmt.Println("[Count] Cache size after clearing the cache:", cache.Count())
}
```

<details>
  <summary>Output</summary>

```
[Count] Cache size: 5
[Get] key=key; value=value; exists=true
[GetByKeys] key=k1; value=v1
[GetByKeys] key=k2; value=v2
[GetByKeys] key=k3; value=v3
[GetKeysByPattern] pattern=key*; key=key-with-ttl
[GetKeysByPattern] pattern=key*; key=key
[TTL] Number of minutes before 'key' expires: 3599
[Count] Cache size after clearing the cache: 0
```
</details>


## Persistence
Prior to v2, gocache supported persistence out of the box.

After some thinking, I decided that persistence added too many dependencies, and given than this is a cache library
and most people wouldn't be interested in persistence, I decided to get rid of it.

That being said, you can use the `GetAll` and `SetAll` methods of `gocache.Cache` to implement persistence yourself.


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
|:-------|:---------|
| goos   | windows  |
| goarch | amd64    |
| cpu    | i7-9700K |
| mem    | 32G DDR4 |

```
// Normal map
BenchmarkMap_Get-8                                                              49944228     24.2 ns/op      7 B/op   0 allocs/op
BenchmarkMap_Set/small_value-8                                                   3939964    394.1 ns/op    188 B/op   2 allocs/op
BenchmarkMap_Set/medium_value-8                                                  3868586    395.5 ns/op    191 B/op   2 allocs/op
BenchmarkMap_Set/large_value-8                                                   3992138    385.3 ns/op    186 B/op   2 allocs/op
// Gocache                                                                                               
BenchmarkCache_Get/FirstInFirstOut-8                                            27907950     44.3 ns/op     7 B/op    0 allocs/op
BenchmarkCache_Get/LeastRecentlyUsed-8                                          28211396     44.2 ns/op     7 B/op    0 allocs/op
BenchmarkCache_Set/FirstInFirstOut_small_value-8                                 3139538    373.5 ns/op    185 B/op   3 allocs/op
BenchmarkCache_Set/FirstInFirstOut_medium_value-8                                3099516    378.6 ns/op    186 B/op   3 allocs/op
BenchmarkCache_Set/FirstInFirstOut_large_value-8                                 3086776    386.7 ns/op    186 B/op   3 allocs/op
BenchmarkCache_Set/LeastRecentlyUsed_small_value-8                               3070555    379.0 ns/op    187 B/op   3 allocs/op
BenchmarkCache_Set/LeastRecentlyUsed_medium_value-8                              3056928    383.8 ns/op    187 B/op   3 allocs/op
BenchmarkCache_Set/LeastRecentlyUsed_large_value-8                               3108250    383.8 ns/op    186 B/op   3 allocs/op
BenchmarkCache_SetUsingMaxMemoryUsage/medium_value-8                             2773315    449.0 ns/op    210 B/op   4 allocs/op
BenchmarkCache_SetUsingMaxMemoryUsage/large_value-8                              2731818    440.0 ns/op    211 B/op   4 allocs/op
BenchmarkCache_SetUsingMaxMemoryUsage/small_value-8                              2659296    446.8 ns/op    213 B/op   4 allocs/op
BenchmarkCache_SetWithMaxSize/100_small_value-8                                  4848658    248.8 ns/op    114 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSize/10000_small_value-8                                4117632    293.7 ns/op    106 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSize/100000_small_value-8                               3867402    313.0 ns/op    110 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSize/100_medium_value-8                                 4750057    250.1 ns/op    113 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSize/10000_medium_value-8                               4143772    294.5 ns/op    106 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSize/100000_medium_value-8                              3768883    313.2 ns/op    111 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSize/100_large_value-8                                  4822646    251.1 ns/op    114 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSize/10000_large_value-8                                4154428    291.6 ns/op    106 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSize/100000_large_value-8                               3897358    313.7 ns/op    110 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/100_small_value-8                            4784180    254.2 ns/op    114 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/10000_small_value-8                          4067042    292.0 ns/op    106 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/100000_small_value-8                         3832760    313.8 ns/op    111 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/100_medium_value-8                           4846706    252.2 ns/op    114 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/10000_medium_value-8                         4103817    292.5 ns/op    106 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/100000_medium_value-8                        3845623    315.1 ns/op    111 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/100_large_value-8                            4744513    257.9 ns/op    114 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/10000_large_value-8                          3956316    299.5 ns/op    106 B/op   3 allocs/op
BenchmarkCache_SetWithMaxSizeAndLRU/100000_large_value-8                         3876843    351.3 ns/op    110 B/op   3 allocs/op
BenchmarkCache_GetSetMultipleConcurrent-8                                         750088   1566.0 ns/op    128 B/op   8 allocs/op
BenchmarkCache_GetSetConcurrentWithFrequentEviction/FirstInFirstOut-8            3836961    316.2 ns/op     80 B/op   1 allocs/op
BenchmarkCache_GetSetConcurrentWithFrequentEviction/LeastRecentlyUsed-8          3846165    315.6 ns/op     80 B/op   1 allocs/op
BenchmarkCache_GetConcurrently/FirstInFirstOut-8                                 4830342    239.8 ns/op      8 B/op   1 allocs/op
BenchmarkCache_GetConcurrently/LeastRecentlyUsed-8                               4895587    243.2 ns/op      8 B/op   1 allocs/op
(Trimmed "BenchmarkCache_" for readability)                                                              
WithForceNilInterfaceOnNilPointer/true_with_nil_struct_pointer-8                 6901461    178.5 ns/op      7 B/op   1 allocs/op
WithForceNilInterfaceOnNilPointer/true-8                                         6629566    180.7 ns/op      7 B/op   1 allocs/op
WithForceNilInterfaceOnNilPointer/false_with_nil_struct_pointer-8                6282798    170.1 ns/op      7 B/op   1 allocs/op
WithForceNilInterfaceOnNilPointer/false-8                                        6741382    172.6 ns/op      7 B/op   1 allocs/op
WithForceNilInterfaceOnNilPointerWithConcurrency/true_with_nil_struct_pointer-8  4432951    258.0 ns/op      8 B/op   1 allocs/op
WithForceNilInterfaceOnNilPointerWithConcurrency/true-8                          4676943    244.4 ns/op      8 B/op   1 allocs/op
WithForceNilInterfaceOnNilPointerWithConcurrency/false_with_nil_struct_pointer-8 4818418    239.6 ns/op      8 B/op   1 allocs/op
WithForceNilInterfaceOnNilPointerWithConcurrency/false-8                         5025937    238.2 ns/op      8 B/op   1 allocs/op
```


## FAQ

### How can I persist the data on application termination?
While creating your own auto save feature might come in handy, it may still lead to loss of data if the application 
automatically saves every 10 minutes and your application crashes 9 minutes after the previous save.

To increase your odds of not losing any data, you can use Go's `signal` package, more specifically its `Notify` function
which allows listening for termination signals like SIGTERM and SIGINT. Once a termination signal is caught, you can
add the necessary logic for a graceful shutdown.

In the following example, the code that would usually be present in the `main` function is moved to a different function
named `Start` which is launched on a different goroutine so that listening for a termination signals is what blocks the
main goroutine instead:
```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/TwiN/gocache/v2"
)

var cache = gocache.NewCache()

func main() {
    data := retrieveCacheEntriesUsingWhateverMeanYouUsedToPersistIt()
    cache.SetAll(data)
    // Start everything else on another goroutine to prevent blocking the main goroutine
    go Start()
    // Wait for termination signal
    sig := make(chan os.Signal, 1)
    done := make(chan bool, 1)
    signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sig
        log.Println("Received termination signal, attempting to gracefully shut down")
        // Persist the cache entries
        cacheEntries := cache.GetAll()
        persistCacheEntriesHoweverYouWant(cacheEntries)
        // Tell the main goroutine that we're done
        done <- true
    }()
    <-done
    log.Println("Shutting down")
}
```

Note that this won't protect you from a SIGKILL, as this signal cannot be caught.
