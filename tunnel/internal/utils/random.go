package utils

import "math/rand"

func GenerateRandomNumbers(start, end, limit int) []int {
	randomNumbers := make([]int, limit)
	for i := range limit {
		randomNumbers[i] = rand.Intn(end-start) + start
	}
	return randomNumbers

}
