from fastapi import APIRouter, Depends
from portr_admin.apis.pagination import PaginatedResponse
from portr_admin.models.user import TeamSettings, TeamUser, User
from portr_admin.apis import security
from portr_admin.schemas.team import (
    AddUserToTeamSchema,
    NewTeamSchema,
    TeamSchema,
    TeamSettingsUpdateSchema,
    TeamSettingsResponseSchema,
)
from portr_admin.schemas.user import TeamUserSchemaForTeam
from portr_admin.services import team as team_service
from portr_admin.utils.exception import PermissionDenied
from portr_admin.config import settings


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


@api.get("/github_events_wh_url")
async def get_team_events_wh_url(
    team_user: TeamUser = Depends(security.requires_admin),
):
    return {
        "url": f"{settings.domain_address()}/api/v1/org_events/?team={team_user.team.slug}"
    }


@api.get("/settings", response_model=TeamSettingsResponseSchema)
async def get_settings(team_user: TeamUser = Depends(security.requires_admin)):
    return (
        await TeamSettings.filter()
        .select_related("updated_by", "updated_by__user")
        .get(team=team_user.team)
    )


@api.patch("/settings", response_model=TeamSettingsResponseSchema)
async def update_settings(
    data: TeamSettingsUpdateSchema,
    team_user: TeamUser = Depends(security.requires_admin),
):
    settings = (
        await TeamSettings.filter()
        .select_related("updated_by", "updated_by__user")
        .get(team=team_user.team)
    )

    settings.auto_invite_github_org_members = data.auto_invite_github_org_members
    if data.github_org_pat is not None:
        settings.github_org_pat = data.github_org_pat  # type: ignore
    if data.github_org_webhook_secret is not None:
        settings.github_org_webhook_secret = data.github_org_webhook_secret  # type: ignore
    settings.updated_by = team_user  # type: ignore

    await settings.save()

    return settings
