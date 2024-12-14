from models.user import TeamUser, User
from tests import TestClient
from tortoise.contrib import test

from tests.factories import TeamUserFactory


class TeamTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.team_user = await TeamUserFactory.create()
        self.client = await TestClient.get_client()
        self.team_user_client = await TestClient.get_logged_in_client(self.team_user)
        self.member_team_user = await TeamUserFactory.create(
            team=self.team_user.team, role="member"
        )
        self.member_team_user_client = await TestClient.get_logged_in_client(
            self.member_team_user
        )
        self.superuser_teamuser = await TeamUserFactory.create(
            team=self.team_user.team, user__is_superuser=True
        )
        self.superuser_teamuser_client = await TestClient.get_logged_in_client(
            self.superuser_teamuser
        )

    async def test_remove_team_user_by_member_should_fail(self):
        team_user_to_delete = await TeamUserFactory.create(team=self.team_user.team)
        resp = self.member_team_user_client.delete(
            f"/api/v1/team/users/{team_user_to_delete.id}",
        )
        assert resp.status_code == 403
        assert resp.json() == {"message": "Only admin can perform this action"}

    async def test_remove_team_user(self):
        team_user_to_delete = await TeamUserFactory.create(team=self.team_user.team)
        resp = self.team_user_client.delete(
            f"/api/v1/team/users/{team_user_to_delete.id}",
        )
        assert resp.status_code == 200
        assert await TeamUser.filter(id=team_user_to_delete.id).first() is None
        assert await User.filter(id=team_user_to_delete.user.id).first() is None

    async def test_remove_superuserteam_user_by_admin_user(self):
        team_user_to_delete = await TeamUserFactory.create(
            team=self.team_user.team, user__is_superuser=True
        )
        resp = self.team_user_client.delete(
            f"/api/v1/team/users/{team_user_to_delete.id}",
        )
        assert resp.status_code == 403
        assert resp.json() == {
            "message": "Only superuser can remove superuser from team"
        }

    async def test_remove_superuserteam_user_by_superuser(self):
        team_user_to_delete = await TeamUserFactory.create(
            team=self.team_user.team, user__is_superuser=True
        )
        resp = self.superuser_teamuser_client.delete(
            f"/api/v1/team/users/{team_user_to_delete.id}",
        )
        assert resp.status_code == 200
        assert await TeamUser.filter(id=team_user_to_delete.id).first() is None
