from fastapi import APIRouter, Depends
from portr_admin.apis.pagination import PaginatedResponse
from portr_admin.models.user import TeamUser, User
from portr_admin.apis import security
from portr_admin.schemas.team import (
    AddUserToTeamSchema,
    NewTeamSchema,
    TeamSchema,
)
from portr_admin.schemas.user import TeamUserSchemaForTeam
from portr_admin.services import team as team_service
from portr_admin.utils.exception import PermissionDenied


api = APIRouter(prefix="/team", tags=["team"])


@api.post("/", response_model=TeamSchema)
async def create_team(
    data: NewTeamSchema, user: User = Depends(security.requires_superuser)
):
    return await team_service.create_team(data.name, user)


@api.get("/users", response_model=PaginatedResponse[TeamUserSchemaForTeam])
async def get_users(
    team_user: TeamUser = Depends(security.get_current_team_user),
    page: int = 1,
    page_size: int = 10,
):
    qs = (
        TeamUser.filter(team=team_user.team)
        .select_related("user", "user__github_user")
        .all()
    )
    return await PaginatedResponse.generate_response_for_page(
        qs=qs, page=page, page_size=page_size
    )


@api.post("/add", response_model=TeamUserSchemaForTeam)
async def add_user(
    data: AddUserToTeamSchema,
    team_user: TeamUser = Depends(security.requires_admin),
):
    if data.set_superuser and not team_user.user.is_superuser:
        raise PermissionDenied("Only superuser can set superuser")

    resp = await team_service.add_user_to_team(
        team=team_user.team,
        email=data.email,
        role=data.role,
        set_superuser=data.set_superuser,
    )
    return resp


@api.delete("/users/{user_id}")
async def remove_user(
    user_id: int,
    team_user: TeamUser = Depends(security.requires_admin),
):
    team_user_to_delete = (
        await TeamUser.filter()
        .select_related("user")
        .get_or_none(id=user_id, team=team_user.team)
    )
    if team_user_to_delete is None:
        raise PermissionDenied("User not found in team")

    if team_user_to_delete.user.is_superuser and not team_user.user.is_superuser:
        raise PermissionDenied("Only superuser can remove superuser from team")

    await TeamUser.filter(id=user_id).delete()

    return {"status": "ok"}
