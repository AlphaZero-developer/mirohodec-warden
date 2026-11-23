package main

func containsInt64(values []int64, needle int64) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
