package trance

func Ternary[T any](condition bool, truthy T, falsy T) T {
	if condition {
		return truthy
	}
	return falsy
}
