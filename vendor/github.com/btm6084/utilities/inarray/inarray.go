package inarray

// Strings returns the position of needle in the haystack if found, -1 otherwise.
func Strings(needle string, haystack []string) int {
	for k, v := range haystack {
		if needle == v {
			return k
		}
	}

	return -1
}

// Ints returns the position of needle in the haystack if found, -1 otherwise.
func Ints(needle int, haystack []int) int {
	for k, v := range haystack {
		if needle == v {
			return k
		}
	}

	return -1
}

// Floats returns the position of needle in the haystack if found, -1 otherwise.
func Floats(needle float64, haystack []float64) int {
	for k, v := range haystack {
		if needle == v {
			return k
		}
	}

	return -1
}
