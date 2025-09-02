package common

func P[T any](v T) *T {
	return &v
}
