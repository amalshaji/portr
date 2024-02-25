from portr_admin.tests import TestClient
from tortoise.contrib import test

from portr_admin.tests.factories import TeamUserFactory, UserFactory


class PageTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.client = await TestClient.get_client()
        self.user = await UserFactory.create()
        self.team_user = await TeamUserFactory.create()
        self.user_auth_client = await TestClient.get_logged_in_client(
            auth_user=self.user
        )
        self.team_user_auth_client = await TestClient.get_logged_in_client(
            auth_user=self.team_user
        )

    def test_root_page_should_pass(self):
        resp = self.client.get("/")
        assert resp.status_code == 200

    def test_new_team_page_not_logged_in_should_redirect_to_root(self):
        resp = self.client.get("/new-team", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == "/?next=%2Fnew-team%3F"

    def test_team_page_not_logged_in_should_redirect_to_root(self):
        resp = self.client.get("/test-team/overview", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == "/?next=%2Ftest-team%2Foverview%3F"

    async def test_team_page_logged_in_should_pass(self):
        resp = self.user_auth_client.get("/new-team", follow_redirects=False)
        assert resp.status_code == 200

    async def test_root_page_logged_in_without_teams_should_redirect_to_new_team_page(
        self,
    ):
        resp = self.user_auth_client.get("/", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == "/new-team"

    async def test_root_page_logged_in_with_teams_should_redirect_to_first_team_overview_page(
        self,
    ):
        resp = self.team_user_auth_client.get("/", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == f"/{self.team_user.team.slug}/overview"
