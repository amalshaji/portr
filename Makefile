buildcli:
	go build -o portr ./cmd/portr

installclient:
	bun --cwd internal/client/dashboard/ui-v2 install

runclient:
	bun --cwd internal/client/dashboard/ui-v2 run dev

buildclient:
	bun --cwd internal/client/dashboard/ui-v2 run build

runclienttestserver:
	cd test-server && uv run main.py
