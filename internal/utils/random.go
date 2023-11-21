package utils

import "math/rand"

func GenerateRandomNumbers(start, end, limit int) []int {
	randomNumbers := make([]int, 100)
	for i := 0; i < 100; i++ {
		randomNumbers[i] = rand.Intn(end-start) + start
	}
	return randomNumbers

}
