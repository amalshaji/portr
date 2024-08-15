from unittest.mock import MagicMock, patch, AsyncMock
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

    @patch("services.user.get_or_create_user")
    @patch("services.auth.login_user")
    async def test_login(self, login_user_fn, get_or_create_user_fn):
        mock_team = MagicMock(slug="test_slug")

        user = MagicMock()
        user.teams = MagicMock()
        user.teams.filter = MagicMock()
        user.teams.filter.return_value.first = AsyncMock(return_value=mock_team)
        get_or_create_user_fn.return_value = user
        resp = self.client.post(
            "/api/v1/auth/login", json={"email": "amal@portr.dev", "password": "amal"}
        )
        assert resp.status_code == 200
        assert resp.json() == {"redirect_to": "/test_slug/overview"}
        get_or_create_user_fn.assert_called_once_with(
            email="amal@portr.dev", password="amal"
        )

    @patch("services.user.get_or_create_user")
    @patch("services.auth.login_user")
    async def test_login_with_wrong_email(self, login_user_fn, get_or_create_user_fn):
        get_or_create_user_fn.side_effect = user_service.UserNotFoundError(
            "User does not exist"
        )
        resp = self.client.post(
            "/api/v1/auth/login", json={"email": "amal@portr.dev", "password": "amal"}
        )
        assert resp.status_code == 400
        assert resp.json() == {"email": "User does not exist"}

    @patch("services.user.get_or_create_user")
    @patch("services.auth.login_user")
    async def test_login_with_wrong_password(
        self, login_user_fn, get_or_create_user_fn
    ):
        get_or_create_user_fn.side_effect = user_service.WrongPasswordError(
            "Password is incorrect"
        )
        resp = self.client.post(
            "/api/v1/auth/login", json={"email": "amal@portr.dev", "password": "amal"}
        )
        assert resp.status_code == 400
        assert resp.json() == {"password": "Password is incorrect"}

    async def test_auth_config(
        self,
    ):
        resp = self.client.get("/api/v1/auth/auth-config")
        assert resp.json() == {"github_auth_enabled": True, "is_first_signup": True}
