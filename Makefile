localportd:
	@go run cmd/localportd/main.go start -c configs/server.yaml

localport:
	@go run cmd/localport/*.go

load-env:
	export $(cat .env | grep -v '^#' | xargs)