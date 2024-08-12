from fastapi.testclient import TestClient as BaseTestClient
from portr_admin.main import app
from portr_admin.models.user import TeamUser, User
from portr_admin.tests.factories import SessionFactory


class TestClient:
    @classmethod
    async def get_client(cls):
        # async so that the signature matches the other method
        return BaseTestClient(app)

    @classmethod
    async def get_logged_in_client(cls, auth_user: User | TeamUser):
        # Separate into two methods?
        if isinstance(auth_user, User):
            user = auth_user
            team_user = None
        else:
            user = auth_user.user
            team_user = auth_user

        client = await cls.get_client()

        session = await SessionFactory.create(user=user)
        client.cookies["portr_session"] = session.token
        if team_user:
            client.headers["x-team-slug"] = team_user.team.slug

        return client
