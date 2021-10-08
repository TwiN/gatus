package gocache

import (
	"fmt"
	"time"
	"unsafe"
)

// Entry is a cache entry
type Entry struct {
	// Key is the name of the cache entry
	Key string

	// Value is the value of the cache entry
	Value interface{}

	// RelevantTimestamp is the variable used to store either:
	// - creation timestamp, if the Cache's EvictionPolicy is FirstInFirstOut
	// - last access timestamp, if the Cache's EvictionPolicy is LeastRecentlyUsed
	//
	// Note that updating an existing entry will also update this value
	RelevantTimestamp time.Time

	// Expiration is the unix time in nanoseconds at which the entry will expire (-1 means no expiration)
	Expiration int64

	next     *Entry
	previous *Entry
}

// Accessed updates the Entry's RelevantTimestamp to now
func (entry *Entry) Accessed() {
	entry.RelevantTimestamp = time.Now()
}

// Expired returns whether the Entry has expired
func (entry Entry) Expired() bool {
	if entry.Expiration > 0 {
		if time.Now().UnixNano() > entry.Expiration {
			return true
		}
	}
	return false
}

// SizeInBytes returns the size of an entry in bytes, approximately.
func (entry *Entry) SizeInBytes() int {
	return toBytes(entry.Key) + toBytes(entry.Value) + 32
}

func toBytes(value interface{}) int {
	switch value.(type) {
	case string:
		return int(unsafe.Sizeof(value)) + len(value.(string))
	case int8, uint8, bool:
		return int(unsafe.Sizeof(value)) + 1
	case int16, uint16:
		return int(unsafe.Sizeof(value)) + 2
	case int32, uint32, float32, complex64:
		return int(unsafe.Sizeof(value)) + 4
	case int64, uint64, int, uint, float64, complex128:
		return int(unsafe.Sizeof(value)) + 8
	case []interface{}:
		size := 0
		for _, v := range value.([]interface{}) {
			size += toBytes(v)
		}
		return int(unsafe.Sizeof(value)) + size
	case []string:
		size := 0
		for _, v := range value.([]string) {
			size += toBytes(v)
		}
		return int(unsafe.Sizeof(value)) + size
	case []int8:
		return int(unsafe.Sizeof(value)) + len(value.([]int8))
	case []uint8:
		return int(unsafe.Sizeof(value)) + len(value.([]uint8))
	case []bool:
		return int(unsafe.Sizeof(value)) + len(value.([]bool))
	case []int16:
		return int(unsafe.Sizeof(value)) + (len(value.([]int16)) * 2)
	case []uint16:
		return int(unsafe.Sizeof(value)) + (len(value.([]uint16)) * 2)
	case []int32:
		return int(unsafe.Sizeof(value)) + (len(value.([]int32)) * 4)
	case []uint32:
		return int(unsafe.Sizeof(value)) + (len(value.([]uint32)) * 4)
	case []float32:
		return int(unsafe.Sizeof(value)) + (len(value.([]float32)) * 4)
	case []complex64:
		return int(unsafe.Sizeof(value)) + (len(value.([]complex64)) * 4)
	case []int64:
		return int(unsafe.Sizeof(value)) + (len(value.([]int64)) * 8)
	case []uint64:
		return int(unsafe.Sizeof(value)) + (len(value.([]uint64)) * 8)
	case []int:
		return int(unsafe.Sizeof(value)) + (len(value.([]int)) * 8)
	case []uint:
		return int(unsafe.Sizeof(value)) + (len(value.([]uint)) * 8)
	case []float64:
		return int(unsafe.Sizeof(value)) + (len(value.([]float64)) * 8)
	case []complex128:
		return int(unsafe.Sizeof(value)) + (len(value.([]complex128)) * 8)
	default:
		return int(unsafe.Sizeof(value)) + len(fmt.Sprintf("%v", value))
	}
}
