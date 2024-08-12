from models.user import TeamUser, User
from tests import TestClient
from tortoise.contrib import test

from tests.factories import TeamUserFactory


class UserTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.team_user = await TeamUserFactory.create()
        self.client = await TestClient.get_client()
        self.user_client = await TestClient.get_logged_in_client(self.team_user.user)
        self.team_user_client = await TestClient.get_logged_in_client(self.team_user)

    async def test_get_current_team_user(self):
        resp = self.team_user_client.get(
            "/api/v1/user/me",
        )
        assert resp.status_code == 200
        assert resp.json() == {
            "id": self.team_user.id,
            "secret_key": self.team_user.secret_key,
            "role": self.team_user.role,
            "user": {
                "email": self.team_user.user.email,
                "first_name": self.team_user.user.first_name,
                "last_name": self.team_user.user.last_name,
                "is_superuser": self.team_user.user.is_superuser,
                "github_user": None,
            },
        }

    async def test_logged_in_user_teams(self):
        resp = self.user_client.get(
            "/api/v1/user/me/teams",
        )
        assert resp.status_code == 200
        assert resp.json() == [
            {
                "id": self.team_user.id,
                "name": self.team_user.team.name,
                "slug": self.team_user.team.slug,
            }
        ]

    async def test_update_logged_in_user(self):
        resp = self.user_client.patch(
            "/api/v1/user/me/update",
            json={"first_name": "test"},
        )
        assert resp.status_code == 200
        assert resp.json() == {
            "email": self.team_user.user.email,
            "first_name": "test",
            "last_name": self.team_user.user.last_name,
            "is_superuser": self.team_user.user.is_superuser,
        }
        assert (await User.get(id=self.team_user.user.id).first()).first_name == "test"

    async def test_rotate_team_user_secret_key(self):
        resp = self.team_user_client.patch("/api/v1/user/me/rotate-secret-key")
        assert resp.status_code == 200
        assert resp.json()["secret_key"] != self.team_user.secret_key
        assert (
            await TeamUser.get(id=self.team_user.id).first()
        ).secret_key == resp.json()["secret_key"]
