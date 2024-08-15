package utils

import (
	"math/rand"
	"time"
)

func GenerateRandomNumbers(start, end, limit int) []int {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))

	randomNumbers := make([]int, limit)
	for i := range limit {
		randomNumbers[i] = rand.Intn(end-start) + start
	}
	return randomNumbers

}
