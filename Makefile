portrd:
	@go run cmd/portrd/main.go start -c configs/server.yaml

portr:
	@go run cmd/portr/*.go

load-env:
	export $(cat .env | grep -v '^#' | xargs)