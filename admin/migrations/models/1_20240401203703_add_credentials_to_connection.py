from tortoise import BaseDBAsyncClient


async def upgrade(db: BaseDBAsyncClient) -> str:
    return """
        ALTER TABLE "connection" ADD "credentials" BYTEA;"""


async def downgrade(db: BaseDBAsyncClient) -> str:
    return """
        ALTER TABLE "connection" DROP COLUMN "credentials";"""
