init-migrations:
	aerich init -t src.portr_admin.db.TORTOISE_ORM

init-db:
	aerich init-db

create-migrations:
	aerich migrate --name $(name)

run-migrations:
	aerich upgrade
