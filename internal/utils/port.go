package utils

func GenerateRandomHttpPorts() []int {
	return GenerateRandomNumbers(20000, 30000, 10)
}

func GenerateRandomTcpPorts() []int {
	return GenerateRandomNumbers(30001, 40001, 10)
}
