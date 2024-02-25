from tortoise import Tortoise, connections
from portr_admin.config import settings

TORTOISE_MODELS = [
    "portr_admin.models.auth",
    "portr_admin.models.user",
    "portr_admin.models.settings",
    "portr_admin.models.connection",
]


async def connect_db(generate_schemas: bool = False):
    await Tortoise.init(
        db_url=settings.db_url,
        modules={"models": TORTOISE_MODELS},
    )
    if generate_schemas:
        await Tortoise.generate_schemas()


async def disconnect_db():
    await connections.close_all()
