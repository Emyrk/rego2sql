package sqlast

type variable struct {
	source RegoSource
	// valueType is only used for typing. Not for the actual value.
	valueType Node
}
