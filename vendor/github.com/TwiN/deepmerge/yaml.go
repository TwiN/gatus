package deepmerge

import (
	"errors"

	"gopkg.in/yaml.v3"
)

var (
	ErrDeepMergeDuplicatePrimitiveKey = errors.New("error deep merging YAML files due to duplicate primitive key: only maps and slices/arrays can be merged, which means you cannot have define the same key twice for parameters that are not maps or slices/arrays")
)

type Config struct {
	// PreventDuplicateKeysWithPrimitiveValue causes YAML to return an error if dst and src define the same key if
	// said key has a value with a primitive type
	// This does not apply to slices or maps. Defaults to true
	PreventDuplicateKeysWithPrimitiveValue bool
}

// YAML merges the contents of src into dst
func YAML(dst, src []byte, optionalConfig ...Config) ([]byte, error) {
	var cfg Config
	if len(optionalConfig) > 0 {
		cfg = optionalConfig[0]
	} else {
		cfg = Config{PreventDuplicateKeysWithPrimitiveValue: true}
	}
	var dstMap, srcMap map[string]interface{}
	err := yaml.Unmarshal(dst, &dstMap)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(src, &srcMap)
	if err != nil {
		return nil, err
	}
	if dstMap == nil {
		dstMap = make(map[string]interface{})
	}
	if err = deepMerge(dstMap, srcMap, cfg); err != nil {
		return nil, err
	}
	return yaml.Marshal(dstMap)
}

func deepMerge(dst, src map[string]interface{}, config Config) error {
	for srcKey, srcValue := range src {
		if srcValueAsMap, ok := srcValue.(map[string]interface{}); ok { // handle maps
			if dstValue, ok := dst[srcKey]; ok {
				if dstValueAsMap, ok := dstValue.(map[string]interface{}); ok {
					err := deepMerge(dstValueAsMap, srcValueAsMap, config)
					if err != nil {
						return err
					}
					continue
				}
			} else {
				dst[srcKey] = make(map[string]interface{})
			}
			err := deepMerge(dst[srcKey].(map[string]interface{}), srcValueAsMap, config)
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
			if config.PreventDuplicateKeysWithPrimitiveValue {
				if _, ok := dst[srcKey]; ok {
					return ErrDeepMergeDuplicatePrimitiveKey
				}
			}
			dst[srcKey] = srcValue
		}
	}
	return nil
}
