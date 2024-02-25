from portr_admin.services import user as user_service
from portr_admin.models.user import Role, Team, TeamUser, User
from tortoise import transactions

from portr_admin.utils.exception import ServiceError
from tortoise.exceptions import IntegrityError


@transactions.atomic()
async def create_team(name: str, user: User) -> Team:
    try:
        team = await Team.create(name=name, owner=user)
    except IntegrityError:
        raise ServiceError("Team with this name already exists")

    _ = await user_service.create_team_user(team, user, Role.admin)
    return team


@transactions.atomic()
async def add_user_to_team(
    team: Team, email: str, role: Role, set_superuser: bool = False
) -> TeamUser:
    user_part_of_team = await TeamUser.filter(team=team, user__email=email).exists()
    if user_part_of_team:
        raise ServiceError("User is already part of the team")

    user, _ = await User.get_or_create(
        email=email, defaults={"is_superuser": set_superuser}
    )
    created_team_user = await user_service.create_team_user(
        team=team, user=user, role=role
    )
    return (
        await TeamUser.filter(id=created_team_user.pk)
        .select_related("user", "user__github_user")
        .first()  # type: ignore
    )
