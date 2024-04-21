from fastapi import Depends, Header, Request

from portr_admin.models.auth import Session
from portr_admin.models.user import Role, TeamUser, User
from portr_admin.services.user import get_or_create_user, is_user_active
from portr_admin.utils.exception import PermissionDenied
from portr_admin.config import settings


class NotAuthenticated(Exception):
    pass


def get_proxy_auth_email(request: Request) -> str | None:
    header = settings.remote_user_header

    if header is None:
        return None

    return request.headers.get(header)

async def get_current_user(
    request: Request
) -> User:
    proxy_auth_email = get_proxy_auth_email(request)

    if proxy_auth_email:
        proxy_auth_user = await get_or_create_user(proxy_auth_email)

        if not is_user_active(proxy_auth_user):
            raise NotAuthenticated

        return proxy_auth_user

    portr_session = request.cookies.get("portr_session")

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
