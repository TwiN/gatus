package deepmerge

import (
	"errors"
)

var (
	ErrKeyWithPrimitiveValueDefinedMoreThanOnce = errors.New("error due to parameter with value of primitive type: only maps and slices/arrays can be merged, which means you cannot have define the same key twice for parameters that are not maps or slices/arrays")
)

func DeepMerge(dst, src map[string]interface{}, config Config) error {
	for srcKey, srcValue := range src {
		if srcValueAsMap, ok := srcValue.(map[string]interface{}); ok { // handle maps
			if dstValue, ok := dst[srcKey]; ok {
				if dstValueAsMap, ok := dstValue.(map[string]interface{}); ok {
					err := DeepMerge(dstValueAsMap, srcValueAsMap, config)
					if err != nil {
						return err
					}
					continue
				}
			} else {
				dst[srcKey] = make(map[string]interface{})
			}
			err := DeepMerge(dst[srcKey].(map[string]interface{}), srcValueAsMap, config)
			if err != nil {
				return err
			}
		} else if srcValueAsSlice, ok := srcValue.([]interface{}); ok { // handle slices
			if dstValue, ok := dst[srcKey]; ok {
				if dstValueAsSlice, ok := dstValue.([]interface{}); ok {
					// If both src and dst are slices, we'll copy the elements from that src slice over to the dst slice
					dst[srcKey] = append(dstValueAsSlice, srcValueAsSlice...)
					continue
				}
			}
			dst[srcKey] = srcValueAsSlice
		} else { // handle primitives
			if config.PreventMultipleDefinitionsOfKeysWithPrimitiveValue {
				if _, ok := dst[srcKey]; ok {
					return ErrKeyWithPrimitiveValueDefinedMoreThanOnce
				}
			}
			dst[srcKey] = srcValue
		}
	}
	return nil
}
