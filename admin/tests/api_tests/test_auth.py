from unittest.mock import patch
from tests import TestClient
from tortoise.contrib import test
from services import user as user_service
from config import settings


class GithubAuthTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.client = await TestClient.get_client()

    @patch("apis.v1.auth.generate_oauth_state")
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
        assert resp.status_code == 307
        assert resp.headers["location"] == "/?code=invalid-state"

    @patch("services.user.get_or_create_user_from_github")
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

    @patch("services.user.get_or_create_user_from_github")
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

    @patch("services.user.get_or_create_user_from_github")
    @patch("services.auth.login_user")
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
