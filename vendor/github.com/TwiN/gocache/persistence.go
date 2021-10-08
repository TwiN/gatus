package gocache

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
)

// SaveToFile stores the content of the cache to a file so that it can be read using
// the ReadFromFile function
func (cache *Cache) SaveToFile(path string) error {
	db, err := bolt.Open(path, os.ModePerm, nil)
	if err != nil {
		return err
	}
	start := time.Now()
	cache.mutex.RLock()
	bulkEntries := make([]*Entry, len(cache.entries))
	i := 0
	for _, v := range cache.entries {
		bulkEntries[i] = v
		i++
	}
	cache.mutex.RUnlock()
	if Debug {
		log.Printf("unlocked after %s", time.Since(start))
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_ = tx.DeleteBucket([]byte("entries"))
		bucket, err := tx.CreateBucket([]byte("entries"))
		if err != nil {
			return err
		}
		for _, bulkEntry := range bulkEntries {
			buffer := bytes.Buffer{}
			err = gob.NewEncoder(&buffer).Encode(bulkEntry)
			if err != nil {
				// Failed to encode the value, so we'll skip it.
				// This is likely due to the fact that the custom struct wasn't registered using gob.Register(...)
				// See [Persistence - Limitations](https://github.com/TwiN/gocache#limitations)
				continue
			}
			bucket.Put([]byte(bulkEntry.Key), buffer.Bytes())
		}
		return nil
	})
	if err != nil {
		return err
	}
	return db.Close()
}

// ReadFromFile populates the cache using a file created using cache.SaveToFile(path)
//
// Note that if the number of entries retrieved from the file exceed the configured maxSize,
// the extra entries will be automatically evicted according to the EvictionPolicy configured.
// This function returns the number of entries evicted, and because this function only reads
// from a file and does not modify it, you can safely retry this function after configuring
// the cache with the appropriate maxSize, should you desire to.
func (cache *Cache) ReadFromFile(path string) (int, error) {
	db, err := bolt.Open(path, os.ModePerm, nil)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("entries"))
		// If the bucket doesn't exist, there's nothing to read, so we'll return right now
		if bucket == nil {
			return nil
		}
		err = bucket.ForEach(func(k, v []byte) error {
			buffer := new(bytes.Buffer)
			decoder := gob.NewDecoder(buffer)
			entry := Entry{}
			buffer.Write(v)
			err := decoder.Decode(&entry)
			if err != nil {
				// Failed to decode the value, so we'll skip it.
				// This is likely due to the fact that the custom struct wasn't registered using gob.Register(...)
				//
				// Could also be due to a breaking change in a struct's variable. For instance, if the struct has
				// a variable with a type map[string]string and that variable is modified to map[string]int,
				// decoding the struct would fail. This can be avoided by using a different variable name every
				// time you must change the type of a variable within a struct.
				//
				// See [Persistence - Limitations](https://github.com/TwiN/gocache#limitations)
				return err
			}
			cache.entries[string(k)] = &entry
			buffer.Reset()
			return nil
		})
		return err
	})
	if err != nil {
		return 0, err
	}
	// Because pointers don't get stored in the file, we need to relink everything from head to tail
	var entries []*Entry
	for _, v := range cache.entries {
		entries = append(entries, v)
	}
	// Sort the slice of entries from oldest to newest
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RelevantTimestamp.Before(entries[j].RelevantTimestamp)
	})
	// Relink the nodes from tail to head
	var previous *Entry
	for i := range entries {
		current := entries[i]
		if previous == nil {
			cache.tail = current
			cache.head = current
		} else {
			previous.previous = current
			current.next = previous
			cache.head = current
		}
		previous = entries[i]
		if cache.maxMemoryUsage != NoMaxMemoryUsage {
			cache.memoryUsage += current.SizeInBytes()
		}
	}
	// If the cache doesn't have a maxSize/maxMemoryUsage, then there's no point checking if we need to evict
	// an entry, so we'll just return now
	if cache.maxSize == NoMaxSize && cache.maxMemoryUsage == NoMaxMemoryUsage {
		return 0, nil
	}
	// Evict what needs to be evicted
	numberOfEvictions := 0
	// If there's a maxSize and the cache has more entries than the maxSize, evict
	if cache.maxSize != NoMaxSize && len(cache.entries) > cache.maxSize {
		for len(cache.entries) > cache.maxSize {
			numberOfEvictions++
			cache.evict()
		}
	}
	// If there's a maxMemoryUsage and the memoryUsage is above the maxMemoryUsage, evict
	if cache.maxMemoryUsage != NoMaxMemoryUsage && cache.memoryUsage > cache.maxMemoryUsage {
		for cache.memoryUsage > cache.maxMemoryUsage && len(cache.entries) > 0 {
			numberOfEvictions++
			cache.evict()
		}
	}
	return numberOfEvictions, nil
}
