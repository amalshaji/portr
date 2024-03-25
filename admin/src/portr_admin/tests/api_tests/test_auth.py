from unittest.mock import patch
from portr_admin.tests import TestClient
from tortoise.contrib import test
from portr_admin.services import user as user_service
from portr_admin.tests.factories import TeamUserFactory, UserFactory
from portr_admin.config import settings


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

    async def test_new_team_page_logged_in_by_normal_user_should_redirect_to_root(self):
        resp = self.user_auth_client.get("/new-team", follow_redirects=False)
        assert resp.status_code == 307
        assert resp.headers["location"] == "/"

    async def test_new_team_page_logged_in_by_super_user_should_pass(self):
        resp = self.superuser_auth_client.get("/new-team", follow_redirects=False)
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


class GithubAuthTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.client = await TestClient.get_client()

    @patch("portr_admin.apis.v1.auth.generate_oauth_state")
    def test_github_login(self, generate_oauth_state_fn):
        generate_oauth_state_fn.return_value = "test_state"

        resp = self.client.get("/api/v1/auth/github", follow_redirects=False)
        assert resp.status_code == 307
        assert (
            resp.headers["location"]
            == f"https://github.com/login/oauth/authorize?client_id={settings.github_client_id}&redirect_uri={settings.domain_address()}/api/v1/auth/github/callback&state=test_state&scope=user:email"
        )
        assert resp.cookies["oauth_state"] == "test_state"

    def test_github_callback_invalid_state(self):
        resp = self.client.get(
            "/api/v1/auth/github/callback?code=test_code&state=invalid_state",
            follow_redirects=False,
        )
        assert resp.status_code == 400
        assert resp.text == "Invalid state"

    @patch("portr_admin.services.user.get_or_create_user_from_github")
    async def test_github_callback_user_not_found(
        self, get_or_create_user_from_github_fn
    ):
        self.client.cookies["oauth_state"] = "test_state"
        get_or_create_user_from_github_fn.side_effect = user_service.UserNotFoundError

        resp = self.client.get(
            "/api/v1/auth/github/callback?code=test_code&state=test_state",
            follow_redirects=False,
        )
        assert resp.status_code == 307
        assert resp.headers["location"] == "/?code=user-not-found"

    @patch("portr_admin.services.user.get_or_create_user_from_github")
    async def test_github_callback_private_email(
        self, get_or_create_user_from_github_fn
    ):
        self.client.cookies["oauth_state"] = "test_state"
        get_or_create_user_from_github_fn.side_effect = user_service.EmailFetchError

        resp = self.client.get(
            "/api/v1/auth/github/callback?code=test_code&state=test_state",
            follow_redirects=False,
        )
        assert resp.status_code == 307
        assert resp.headers["location"] == "/?code=private-email"

    @patch("portr_admin.services.user.get_or_create_user_from_github")
    @patch("portr_admin.services.auth.login_user")
    async def test_github_callback(
        self, login_user_fn, get_or_create_user_from_github_fn
    ):
        self.client.cookies["oauth_state"] = "test_state"
        login_user_fn.return_value = "test_token"

        resp = self.client.get(
            "/api/v1/auth/github/callback?code=test_code&state=test_state",
            follow_redirects=False,
        )
        assert resp.status_code == 307
        assert resp.headers["location"] == "/"
        login_user_fn.assert_called_once()
        get_or_create_user_from_github_fn.assert_called_once()
        get_or_create_user_from_github_fn.assert_called_with("test_code")
        login_user_fn.assert_called_with(get_or_create_user_from_github_fn.return_value)
