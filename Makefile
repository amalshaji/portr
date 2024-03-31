load-env:
	export $(cat .env | grep -v '^#' | xargs)

build-push-main:
	docker build -t amalshaji/portr-admin:main ./admin
	docker image push docker.io/amalshaji/portr-admin:main

	docker build -t amalshaji/portr-tunnel:main ./tunnel
	docker image push docker.io/amalshaji/portr-tunnel:main