buildcli:
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o portr ./cmd/portr

installclient:
	pnpm --dir internal/client/dashboard/ui install

runclient:
	pnpm --dir internal/client/dashboard/ui dev

buildclient:
	pnpm --dir internal/client/dashboard/ui build
