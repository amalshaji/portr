package utils

import "math/rand"

func GenerateRandomNumbers(start, end, limit int) []int {
	randomNumbers := make([]int, 100)
	for i := range 100 {
		randomNumbers[i] = rand.Intn(end-start) + start
	}
	return randomNumbers

}
