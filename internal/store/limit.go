package store

func Limit[T any](items []T, limit int) []T {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}
