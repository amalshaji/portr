package utils

import (
	"math/rand"
	"time"
)

// GenerateRandomNumbers returns up to limit distinct numbers in [start, end).
// Values are unique so callers (e.g. port allocation) never get a duplicate.
func GenerateRandomNumbers(start, end, limit int) []int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	span := end - start
	if limit > span {
		limit = span
	}

	seen := make(map[int]bool, limit)
	randomNumbers := make([]int, 0, limit)
	for len(randomNumbers) < limit {
		n := r.Intn(span) + start
		if seen[n] {
			continue
		}
		seen[n] = true
		randomNumbers = append(randomNumbers, n)
	}
	return randomNumbers
}
