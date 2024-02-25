from fastapi import APIRouter, Depends

from portr_admin.apis import security
from portr_admin.apis.pagination import PaginatedResponse
from portr_admin.enums import Enum
from portr_admin.models.connection import Connection, ConnectionStatus
from portr_admin.services import user as user_service
from portr_admin.models.user import TeamUser
from portr_admin.schemas.connection import ConnectionCreateSchema, ConnectionSchema
from portr_admin.utils.exception import ServiceError
from portr_admin.services import connection as connection_service

api = APIRouter(prefix="/connections", tags=["connections"])


class ConnectionQueryType(Enum):
    active = "active"
    recent = "recent"


@api.get("/", response_model=PaginatedResponse[ConnectionSchema])
async def get_connections(
    team_user: TeamUser = Depends(security.get_current_team_user),
    type: ConnectionQueryType = ConnectionQueryType.recent,
    page: int = 1,
    page_size: int = 10,
):
    qs = (
        Connection.filter(team=team_user.team)
        .select_related("created_by", "team")
        .prefetch_related("created_by__user")
        .order_by("-created_at")
    )
    if type == ConnectionQueryType.active:
        qs = qs.filter(status=ConnectionStatus.active.value)

    return await PaginatedResponse.generate_response_for_page(
        qs=qs.all(), page=page, page_size=page_size
    )


@api.post("/")
async def create_connection(data: ConnectionCreateSchema):
    team_user = await user_service.get_team_user_by_secret_key(data.secret_key)
    if not team_user:
        raise ServiceError("Invalid secret key")

    connection = await connection_service.create_new_connection(
        type=data.connection_type,  # type: ignore
        subdomain=data.subdomain,
        created_by=team_user,
    )
    return {"connection_id": connection.id}
