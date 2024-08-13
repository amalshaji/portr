init-migrations:
	aerich init -t db.TORTOISE_ORM

init-db:
	aerich init-db

create-migrations:
	aerich migrate --name $(name)

run-migrations:
	aerich upgrade

runserver:
	make run-migrations
	uvicorn main:app --reload

installclient:
	pnpm --dir web install

runclient:
	pnpm --dir web dev