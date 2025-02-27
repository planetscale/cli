package testutil

func Pointer[K any](val K) *K {
	return &val
}
