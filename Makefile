buildcli:
	go build -o portr ./cmd/portr

gopackages:
	./scripts/go-packages.sh

buildgo:
	go build $$(./scripts/go-packages.sh --build)

testgo:
	go test $$(./scripts/go-packages.sh)

installclient:
	bun --cwd internal/client/dashboard/ui-v2 install

runclient:
	bun --cwd internal/client/dashboard/ui-v2 run dev

buildclient:
	bun --cwd internal/client/dashboard/ui-v2 run build

runclienttestserver:
	cd test-server && uv run main.py
