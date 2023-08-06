package stdlib

// Returns true if the given value is in the slice.
func InSlice[T comparable](slice *[]T, val T) bool {
	for _, v := range *slice {
		if v == val {
			return true
		}
	}
	return false
}
