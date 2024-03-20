from datetime import datetime, timedelta, UTC
from portr_admin.models.auth import Session
from portr_admin.models.connection import Connection, ConnectionStatus
import logging

logger = logging.getLogger("fastapi")


async def clear_expired_sessions():
    logger.info("Clearing expired sessions")
    await Session.filter(expires_at__lte=datetime.utcnow()).delete()


async def clear_unclaimed_connections():
    logger.info(f"{datetime.now(UTC)} Clearing unclaimed connections")
    await Connection.filter(
        status=ConnectionStatus.reserved.value,
        created_at__lte=datetime.now(UTC) - timedelta(seconds=10),
    ).delete()
