init-migrations:
	aerich init -t src.portr_admin.db.TORTOISE_ORM

init-db:
	aerich init-db

create-migrations:
	aerich migrate --name $(name)

run-migrations:
	aerich upgrade

runserver:
	make run-migrations
	uvicorn src.portr_admin.main:app --reload

installclient:
	pnpm --dir src/web install

runclient:
	pnpm --dir src/web dev