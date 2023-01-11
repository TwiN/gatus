package deepmerge

type Config struct {
	// PreventMultipleDefinitionsOfKeysWithPrimitiveValue causes the return of an error if dst and src define
	// the same key and if said key has a value with a primitive type
	// This does not apply to slices or maps.
	//
	// Defaults to true
	PreventMultipleDefinitionsOfKeysWithPrimitiveValue bool
}
