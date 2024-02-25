package utils

func GenerateRandomHttpPorts() []int {
	var startPort int = 20000
	var endPort int = 30000

	return GenerateRandomNumbers(startPort, endPort, 100)
}

func GenerateRandomTcpPorts() []int {
	var startPort int = 30001
	var endPort int = 40001

	return GenerateRandomNumbers(startPort, endPort, 100)
}
