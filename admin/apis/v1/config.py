import asyncio
from fastapi import APIRouter, Depends, Request
from pydantic import BaseModel
from apis import security
from config import settings
from models.connection import Connection
from models.user import TeamUser
from utils.exception import ServiceError
from services import user as user_service


api = APIRouter(prefix="/config", tags=["config"])


class GetConfigInput(BaseModel):
    secret_key: str


DEFAULT_CONFIG = """
server_url: {server_url}
ssh_url: {ssh_url}
secret_key: {secret_key}
enable_request_logging: false
connection_log_retention_days: 0
tunnels:
  - name: portr
    subdomain: portr
    port: 4321
""".strip()

SETUP_SCRIPT = """
portr auth set --token {token} --remote {server_url}
""".strip()


@api.post("/download")
async def download_config(data: GetConfigInput):
    team_user = await user_service.get_team_user_by_secret_key(data.secret_key)
    if not team_user:
        raise ServiceError("Invalid secret key")

    return {
        "message": DEFAULT_CONFIG.format(
            server_url=settings.server_url,
            ssh_url=settings.ssh_url,
            secret_key=team_user.secret_key,
        )
    }


@api.get("/setup-script")
async def setup_script(team_user: TeamUser = Depends(security.get_current_team_user)):
    return {
        "message": SETUP_SCRIPT.format(
            token=team_user.secret_key, server_url=settings.server_url
        )
    }


@api.get("/stats")
async def get_stats(
    request: Request, team_user: TeamUser = Depends(security.get_current_team_user)
):
    team = team_user.team

    total_connections_query = Connection.filter(team_id=team.id).count()
    active_connections_query = Connection.filter(
        team_id=team.id, status="active"
    ).count()
    team_members_count = TeamUser.filter(team_id=team.id).count()

    total_connections, active_connections, team_members = await asyncio.gather(
        total_connections_query, active_connections_query, team_members_count
    )

    return {
        "team_stats": {
            "total_connections": total_connections,
            "active_connections": active_connections,
            "team_members": team_members,
        },
        "system_stats": {
            "server_start_time": request.app.state.server_start_time,
        },
    }
