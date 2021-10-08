package gocache

import (
	"log"
	"time"
)

const (
	// JanitorShiftTarget is the target number of expired keys to find during passive clean up duty
	// before pausing the passive expired keys eviction process
	JanitorShiftTarget = 25

	// JanitorMaxIterationsPerShift is the maximum number of nodes to traverse before pausing
	JanitorMaxIterationsPerShift = 1000

	// JanitorMinShiftBackOff is the minimum interval between each iteration of steps
	// defined by JanitorMaxIterationsPerShift
	JanitorMinShiftBackOff = time.Millisecond * 50

	// JanitorMaxShiftBackOff is the maximum interval between each iteration of steps
	// defined by JanitorMaxIterationsPerShift
	JanitorMaxShiftBackOff = time.Millisecond * 500
)

// StartJanitor starts the janitor on a different goroutine
// The janitor's job is to delete expired keys in the background, in other words, it takes care of passive eviction.
// It can be stopped by calling Cache.StopJanitor.
// If you do not start the janitor, expired keys will only be deleted when they are accessed through Get, GetByKeys, or
// GetAll.
func (cache *Cache) StartJanitor() error {
	if cache.stopJanitor != nil {
		return ErrJanitorAlreadyRunning
	}
	cache.stopJanitor = make(chan bool)
	go func() {
		// rather than starting from the tail on every run, we can try to start from the last traversed entry
		var lastTraversedNode *Entry
		totalNumberOfExpiredKeysInPreviousRunFromTailToHead := 0
		backOff := JanitorMinShiftBackOff
		for {
			select {
			case <-time.After(backOff):
				// Passive clean up duty
				cache.mutex.Lock()
				if cache.tail != nil {
					start := time.Now()
					steps := 0
					expiredEntriesFound := 0
					current := cache.tail
					if lastTraversedNode != nil {
						// Make sure the lastTraversedNode is still in the cache, otherwise we might be traversing nodes that were already deleted.
						// Furthermore, we need to make sure that the entry from the cache has the same pointer as the lastTraversedNode
						// to verify that there isn't just a new cache entry with the same key (i.e. in case lastTraversedNode got evicted)
						if entryFromCache, isInCache := cache.get(lastTraversedNode.Key); isInCache && entryFromCache == lastTraversedNode {
							current = lastTraversedNode
						}
					}
					if current == cache.tail {
						if Debug {
							log.Printf("There are currently %d entries in the cache. The last walk resulted in finding %d expired keys", len(cache.entries), totalNumberOfExpiredKeysInPreviousRunFromTailToHead)
						}
						totalNumberOfExpiredKeysInPreviousRunFromTailToHead = 0
					}
					for current != nil {
						// since we're walking from the tail to the head, we get the previous reference
						var previous *Entry
						steps++
						if current.Expired() {
							expiredEntriesFound++
							// Because delete will remove the previous reference from the entry, we need to store the
							// previous reference before we delete it
							previous = current.previous
							cache.delete(current.Key)
							cache.stats.ExpiredKeys++
						}
						if current == cache.head {
							lastTraversedNode = nil
							break
						}
						// Travel to the current node's previous node only if no specific previous node has been specified
						if previous != nil {
							current = previous
						} else {
							current = current.previous
						}
						lastTraversedNode = current
						if steps == JanitorMaxIterationsPerShift || expiredEntriesFound >= JanitorShiftTarget {
							if expiredEntriesFound > 0 {
								backOff = JanitorMinShiftBackOff
							} else {
								if backOff*2 <= JanitorMaxShiftBackOff {
									backOff *= 2
								} else {
									backOff = JanitorMaxShiftBackOff
								}
							}
							break
						}
					}
					if Debug {
						log.Printf("traversed %d nodes and found %d expired entries in %s before stopping\n", steps, expiredEntriesFound, time.Since(start))
					}
					totalNumberOfExpiredKeysInPreviousRunFromTailToHead += expiredEntriesFound
				} else {
					if backOff*2 < JanitorMaxShiftBackOff {
						backOff *= 2
					} else {
						backOff = JanitorMaxShiftBackOff
					}
				}
				cache.mutex.Unlock()
			case <-cache.stopJanitor:
				cache.stopJanitor <- true
				return
			}
		}
	}()
	//if Debug {
	//	go func() {
	//		var m runtime.MemStats
	//		for {
	//			runtime.ReadMemStats(&m)
	//			log.Printf("Alloc=%vMB; HeapReleased=%vMB; Sys=%vMB; HeapInUse=%vMB; HeapObjects=%v; HeapObjectsFreed=%v; GC=%v; cache.memoryUsage=%vMB; cacheSize=%d\n", m.Alloc/1024/1024, m.HeapReleased/1024/1024, m.Sys/1024/1024, m.HeapInuse/1024/1024, m.HeapObjects, m.Frees, m.NumGC, cache.memoryUsage/1024/1024, cache.Count())
	//			time.Sleep(3 * time.Second)
	//		}
	//	}()
	//}
	return nil
}

// StopJanitor stops the janitor
func (cache *Cache) StopJanitor() {
	if cache.stopJanitor != nil {
		// Tell the janitor to stop, and then wait for the janitor to reply on the same channel that it's stopping
		// This may seem a bit odd, but this allows us to avoid a data race condition when trying to set
		// cache.stopJanitor to nil
		cache.stopJanitor <- true
		<-cache.stopJanitor
		cache.stopJanitor = nil
	}
}
