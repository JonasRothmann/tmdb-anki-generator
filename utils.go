package tmdbankigenerator

func Ptr[T any](v T) *T {
	return &v
}
