from portr_admin.tests import TestClient
from tortoise.contrib import test
from portr_admin.tests.factories import TeamUserFactory, UserFactory


class PageTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.client = await TestClient.get_client()
        self.user = await UserFactory.create()
        self.superuser_user = await UserFactory.create(is_superuser=True)
        self.team_user = await TeamUserFactory.create()
        self.user_auth_client = await TestClient.get_logged_in_client(
            auth_user=self.user
        )
        self.superuser_auth_client = await TestClient.get_logged_in_client(
            auth_user=self.superuser_user
        )
        self.team_user_auth_client = await TestClient.get_logged_in_client(
            auth_user=self.team_user
        )

    def test_root_page(self):
        resp = self.client.get("/", follow_redirects=False)
        assert resp.status_code == 200

    async def test_root_page_logged_in_with_no_team(self):
        resp = self.user_auth_client.get("/", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == "/instance-settings/team"

    async def test_root_page_logged_in_with_team(self):
        team_user = await TeamUserFactory.create(user=self.user)
        resp = self.user_auth_client.get("/", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == f"/{team_user.team.slug}/overview"

    def test_email_settings_page_not_logged_in_should_redirect_to_root(self):
        resp = self.client.get("/instance-settings/email", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == "/?next=%2Finstance-settings%2Femail%3F"

    def test_team_settings_page_not_logged_in_should_redirect_to_root(self):
        resp = self.client.get("/instance-settings/team", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == "/?next=%2Finstance-settings%2Fteam%3F"

    def test_instance_settings_page_as_regular_user_should_redirect_to_root(self):
        resp = self.user_auth_client.get(
            "/instance-settings/email", follow_redirects=False
        )
        assert resp.status_code == 307
        assert resp.headers["location"] == "/"

    def test_instance_settings_page_as_super_user_should_redirect_to_root(self):
        resp = self.superuser_auth_client.get(
            "/instance-settings/email", follow_redirects=False
        )
        assert resp.status_code == 200

    def test_team_page_not_logged_in_should_redirect_to_root(self):
        resp = self.client.get("/test-team/overview", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == "/?next=%2Ftest-team%2Foverview%3F"

    async def test_diff_team_page_logged_in_should_redirect_to_first_team(self):
        resp = self.team_user_auth_client.get(
            "/test-team/overview", follow_redirects=False
        )
        assert resp.status_code == 307
        assert resp.headers["location"] == "/"

    async def test_root_page_logged_in_with_teams_should_redirect_to_first_team_overview_page(
        self,
    ):
        resp = self.team_user_auth_client.get("/", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == f"/{self.team_user.team.slug}/overview"
