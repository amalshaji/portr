from portr_admin.tests import TestClient
from tortoise.contrib import test

from portr_admin.tests.factories import (
    TeamSettingsFactory,
    TeamUserFactory,
)


class ConnectionTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.settings = await TeamSettingsFactory.create()
        self.team_user = await TeamUserFactory.create(team=self.settings.team)
        self.member_team_user = await TeamUserFactory.create(
            team=self.settings.team, role="member"
        )
        self.team_user_client = await TestClient.get_logged_in_client(self.team_user)
        self.member_team_user_client = await TestClient.get_logged_in_client(
            self.member_team_user
        )

    async def test_get_settings_should_pass(self):
        resp = self.team_user_client.get("/api/v1/team/settings")
        assert resp.status_code == 200

        data = resp.json()

        assert not data["auto_invite_github_org_members"]
        assert data["github_org_pat"] is None
        assert data["github_org_webhook_secret"] is None

    async def test_get_settings_by_member_should_pass(self):
        resp = self.member_team_user_client.get("/api/v1/team/settings")
        assert resp.status_code == 403
        assert resp.json() == {"message": "Only admin can perform this action"}

    async def test_update_settings_should_pass(self):
        resp = self.team_user_client.patch(
            "/api/v1/team/settings",
            json={
                "auto_invite_github_org_members": True,
                "github_org_pat": "test-pat",
            },
        )
        assert resp.status_code == 200

        data = resp.json()

        assert data["auto_invite_github_org_members"]
        assert data["github_org_pat"] == "test-pat"
