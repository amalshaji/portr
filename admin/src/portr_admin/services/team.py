from portr_admin.models.settings import InstanceSettings
from portr_admin.services import user as user_service
from portr_admin.services import settings as settings_service
from portr_admin.models.user import Role, Team, TeamSettings, TeamUser, User
from tortoise import transactions
from portr_admin.config import settings
from portr_admin.utils.exception import ServiceError
from tortoise.exceptions import IntegrityError
from portr_admin.utils import smtp
from portr_admin.utils.template_renderer import render_template


@transactions.atomic()
async def create_team(name: str, user: User) -> Team:
    try:
        team = await Team.create(name=name, owner=user)
    except IntegrityError:
        raise ServiceError("Team with this name already exists")

    _ = await user_service.create_team_user(team, user, Role.admin)
    _ = await TeamSettings.create(team=team)
    return team


async def send_notification(email: str, team: Team, global_settings: InstanceSettings):
    context = {
        "teamName": team.name,
        "email": email,
        "appUrl": settings.domain_address(),
        "dashboardUrl": f"{settings.domain_address()}/{team.name}/overview",
    }

    subject = render_template(global_settings.add_user_email_subject, context)
    body = render_template(global_settings.add_user_email_body, context)

    await smtp.send_mail(to=email, subject=subject, body=body)


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
    team_user = (
        await TeamUser.filter(id=created_team_user.pk)
        .select_related("user", "user__github_user")
        .first()  # type: ignore
    )

    global_settings = await settings_service.get_instance_settings()

    if global_settings.smtp_enabled:
        await send_notification(email, team, global_settings)

    return team_user  # type: ignore
