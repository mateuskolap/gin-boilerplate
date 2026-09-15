package collection

func Map[T any, R any](items []T, iteratee func(T) R) []R {
	result := make([]R, 0, len(items))
	for _, item := range items {
		result = append(result, iteratee(item))
	}

	return result
}
