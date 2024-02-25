from portr_admin.models.user import GithubUser, Role, Team, TeamUser, User
from portr_admin.utils.github_auth import GithubOauth
from portr_admin.config import settings
from tortoise import transactions


@transactions.atomic()
async def get_or_create_user_from_github(code: str):
    client = GithubOauth(
        client_id=settings.github_app_client_id,
        client_secret=settings.github_app_client_secret,
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

    is_superuser = await User.filter().count() == 0

    user, _ = await User.get_or_create(
        email=github_user["email"],
        defaults={"is_superuser": is_superuser},
    )

    github_user_obj, created = await GithubUser.get_or_create(
        user=user,
        defaults={
            "github_access_token": token,
            "github_avatar_url": github_user["avatar_url"],
        },
    )

    if not created:
        github_user_obj.github_access_token = token
        github_user_obj.github_avatar_url = github_user["avatar_url"]
        await github_user_obj.save()

    return user


async def create_team_user(team: Team, user: User, role: Role) -> TeamUser:
    return await TeamUser.create(team=team, user=user, role=role.value)


async def get_team_user_by_secret_key(secret_key: str) -> TeamUser | None:
    return await TeamUser.filter(secret_key=secret_key).select_related("team").first()
