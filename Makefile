buildcli:
	go build -o portr ./cmd/portr

gopackages:
	./scripts/go-packages.sh

buildgo:
	go build $$(./scripts/go-packages.sh --build)

testgo:
	go test $$(./scripts/go-packages.sh)

installclient:
	bun install --cwd internal/client/dashboard/ui-v2

runclient:
	bun run --cwd internal/client/dashboard/ui-v2 dev

buildclient:
	bun run --cwd internal/client/dashboard/ui-v2 build

runclienttestserver:
	cd test-server && uv run main.py
