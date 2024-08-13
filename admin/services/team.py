from models.settings import InstanceSettings
from services import user as user_service
from services import settings as settings_service
from models.user import Role, Team, TeamUser, User
from tortoise import transactions
from config import settings
from utils.exception import ServiceError
from tortoise.exceptions import IntegrityError
from utils import smtp
from utils.template_renderer import render_template
from utils.token import generate_random_password


@transactions.atomic()
async def create_team(name: str, user: User) -> Team:
    try:
        team = await Team.create(name=name, owner=user)
    except IntegrityError:
        raise ServiceError("Team with this name already exists")

    _ = await user_service.create_team_user(team=team, user=user, role=Role.admin)
    return team


DEFAULT_TEAM_NAME = "Portr"


async def create_default_team(user: User) -> Team:
    return await create_team(DEFAULT_TEAM_NAME, user)


async def send_notification(
    email: str, team: Team, instance_settings: InstanceSettings
):
    context = {
        "teamName": team.name,
        "email": email,
        "appUrl": settings.domain_address(),
        "dashboardUrl": f"{settings.domain_address()}/{team.slug}/overview",
    }

    subject = render_template(instance_settings.add_user_email_subject, context)
    body = render_template(instance_settings.add_user_email_body, context)

    await smtp.send_mail(to=email, subject=subject, body=body)


@transactions.atomic()
async def add_user_to_team(
    team: Team, email: str, role: Role, set_superuser: bool = False
) -> tuple[TeamUser, str]:
    user_part_of_team = await TeamUser.filter(team=team, user__email=email).exists()
    if user_part_of_team:
        raise ServiceError("User is already part of the team")

    user, created = await User.get_or_create(
        email=email, defaults={"is_superuser": set_superuser}
    )

    password = None
    if created:
        password = generate_random_password()
        user.set_password(password)
        await user.save()

    created_team_user = await user_service.create_team_user(
        team=team, user=user, role=role
    )
    team_user = (
        await TeamUser.filter(id=created_team_user.pk)
        .select_related("user", "user__github_user")
        .first()  # type: ignore
    )

    instance_settings = await settings_service.get_instance_settings()

    if instance_settings.smtp_enabled:
        await send_notification(email, team, instance_settings)

    return team_user, password  # type: ignore
