from portr_admin.models.user import GithubUser, Role, Team, TeamUser, User
from portr_admin.utils.exception import ServiceError
from portr_admin.utils.github_auth import GithubOauth
from portr_admin.config import settings
from tortoise import transactions
from tortoise.exceptions import ValidationError


class UserNotFoundError(ServiceError):
    pass


class EmailFetchError(ServiceError):
    pass


@transactions.atomic()
async def get_or_create_user(email: str):
    has_users = await User.filter().exists()

    if not has_users:
        # If this is the first user we can create them as a superuser
        return await User.create(
            email=email,
            is_superuser=True,
        )

    user = await User.get_or_none(email=email)

    if not user:
        raise UserNotFoundError("User does not exist")

    return user


def is_user_active(user: User):
    """
    A user is considered "active" if they're on a team or
    if they're a superuser.  An inactive user cannot authenticate.
    """
    is_user_on_team = len(user.teams) > 0

    return user.is_superuser or is_user_on_team


@transactions.atomic()
async def get_or_create_user_from_github(code: str):
    client = GithubOauth(
        client_id=settings.github_client_id,
        client_secret=settings.github_client_secret,
    )
    token = await client.get_access_token(code)
    github_user = await client.get_user(token)

    # if the user emails are private, we need to get the emails
    # pick the first verified and primary email
    if not github_user["email"]:
        emails = await client.get_emails(token)
        for email in emails:
            if email["verified"] and email["primary"]:
                github_user["email"] = email["email"]
                break

    if not github_user["email"]:
        raise EmailFetchError("No verified email found")

    user = await get_or_create_user(github_user["email"])

    github_user_obj, created = await GithubUser.get_or_create(
        user=user,
        defaults={
            "github_id": github_user["id"],
            "github_access_token": token,
            "github_avatar_url": github_user["avatar_url"],
        },
    )

    if not created:
        github_user_obj.github_id = github_user["id"]
        github_user_obj.github_access_token = token
        github_user_obj.github_avatar_url = github_user["avatar_url"]
        await github_user_obj.save()

    return user


async def create_team_user(team: Team, user: User, role: Role) -> TeamUser:
    return await TeamUser.create(team=team, user=user, role=role.value)


async def get_team_user_by_secret_key(secret_key: str) -> TeamUser | None:
    try:
        return (
            await TeamUser.filter(secret_key=secret_key).select_related("team").first()
        )
    except ValidationError:
        raise ServiceError("Invalid secret key")
