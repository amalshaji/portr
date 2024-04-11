from portr_admin.models.connection import Connection, ConnectionStatus, ConnectionType
from portr_admin.models.user import TeamUser
from portr_admin.utils.exception import ServiceError


async def create_new_connection(
    type: ConnectionType,
    created_by: TeamUser,
    subdomain: str | None = None,
    port: int | None = None,
    credentials: str | None = None,
) -> Connection:
    if type == ConnectionType.http and not subdomain:
        raise ServiceError("subdomain is required for http connections")

    if type == ConnectionType.http:
        active_connection = await Connection.filter(
            subdomain=subdomain,
            status=ConnectionStatus.active.value,  # type: ignore
        ).first()
        if active_connection:
            raise ServiceError("Subdomain already in use")

    connection = await Connection.create(
        type=type,
        subdomain=subdomain if type == ConnectionType.http else None,
        port=port if type == ConnectionType.tcp else None,
        created_by=created_by,
        team=created_by.team,
    )

    if credentials:
        connection.credentials = credentials  # type: ignore
        await connection.save()

    return connection
