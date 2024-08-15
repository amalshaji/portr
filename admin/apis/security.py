from typing import Annotated
from fastapi import Cookie, Depends, Header

from models.auth import Session
from models.user import Role, TeamUser, User

from utils.exception import PermissionDenied


class NotAuthenticated(Exception):
    pass


async def get_current_user(
    portr_session: Annotated[str | None, Cookie()] = None,
) -> User:
    if portr_session is None:
        raise NotAuthenticated

    session = await Session.filter(token=portr_session).select_related("user").first()
    if session is None:
        raise NotAuthenticated

    return session.user


async def get_current_team_user(
    user: User = Depends(get_current_user),
    x_team_slug: str | None = Header(),
) -> TeamUser:
    if x_team_slug is None:
        raise NotAuthenticated

    team_user = (
        await TeamUser.filter(user=user, team__slug=x_team_slug)
        .select_related("team", "user", "user__github_user")
        .first()
    )
    if team_user is None:
        raise NotAuthenticated

    return team_user


async def requires_superuser(user: User = Depends(get_current_user)) -> User:
    if not user.is_superuser:
        raise PermissionDenied("Only superuser can perform this action")

    return user


async def requires_admin(
    team_user: TeamUser = Depends(get_current_team_user),
) -> TeamUser:
    if team_user.role != Role.admin:
        raise PermissionDenied("Only admin can perform this action")

    return team_user
