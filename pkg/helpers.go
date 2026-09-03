package pkg

func Map[T any, R any](collection []T, iteratee func(T) R) []R {
	result := make([]R, 0, len(collection))
	for _, item := range collection {
		result = append(result, iteratee(item))
	}

	return result
}
