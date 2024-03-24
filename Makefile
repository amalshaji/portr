load-env:
	export $(cat .env | grep -v '^#' | xargs)