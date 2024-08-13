from fastapi import APIRouter, Depends

from apis import security
from apis.pagination import PaginatedResponse
from config.enums import Enum
from models.connection import Connection, ConnectionStatus
from services import user as user_service
from models.user import TeamUser
from schemas.connection import ConnectionCreateSchema, ConnectionSchema
from utils.exception import ServiceError
from services import connection as connection_service

api = APIRouter(prefix="/connections", tags=["connections"])


class ConnectionQueryType(Enum):
    active = "active"
    recent = "recent"


@api.get("/", response_model=PaginatedResponse[ConnectionSchema])
async def get_connections(
    team_user: TeamUser = Depends(security.get_current_team_user),
    type: ConnectionQueryType = ConnectionQueryType.recent,  # type: ignore
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
