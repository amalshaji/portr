from datetime import datetime, timedelta, UTC
from models.auth import Session
from models.connection import Connection, ConnectionStatus
import logging

logger = logging.getLogger("uvicorn.info")


async def clear_expired_sessions():
    logger.info("Clearing expired sessions")
    await Session.filter(expires_at__lte=datetime.now(UTC)).delete()


async def clear_unclaimed_connections():
    logger.info("Clearing unclaimed connections")
    await Connection.filter(
        status=ConnectionStatus.reserved.value,
        created_at__lte=datetime.now(UTC) - timedelta(seconds=10),
    ).delete()
