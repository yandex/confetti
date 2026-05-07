package ptr

// T returns pointer to provided value
func T[T any](v T) *T { return &v }
