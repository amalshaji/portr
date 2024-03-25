import asyncio
from aerich import Command  # type: ignore

from portr_admin.db import TORTOISE_ORM, connect_db, disconnect_db
from portr_admin.services.settings import populate_instance_settings


command = Command(tortoise_config=TORTOISE_ORM)


async def run_migrations():
    await command.init()
    await command.upgrade(run_in_transaction=True)


async def populate_settings():
    await connect_db()
    await populate_instance_settings()
    await disconnect_db()


async def main():
    await run_migrations()
    await populate_settings()


asyncio.run(main())
