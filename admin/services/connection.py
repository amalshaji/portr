from models.connection import Connection, ConnectionStatus, ConnectionType
from models.user import TeamUser
from utils.exception import ServiceError


async def create_new_connection(
    type: ConnectionType,
    created_by: TeamUser,
    subdomain: str | None = None,
    port: int | None = None,
) -> Connection:
    if type == ConnectionType.http and not subdomain:
        raise ServiceError("subdomain is required for http connections")

    if type == ConnectionType.http:
        active_connection = await Connection.filter(
            subdomain=subdomain, status=ConnectionStatus.active.value
        ).first()
        if active_connection:
            raise ServiceError("Subdomain already in use")

    return await Connection.create(
        type=type,
        subdomain=subdomain if type == ConnectionType.http else None,
        port=port if type == ConnectionType.tcp else None,
        created_by=created_by,
        team=created_by.team,
    )
