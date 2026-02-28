from tests import TestClient
from tortoise.contrib import test
from config import settings
from tests.factories import TeamUserFactory, UserFactory


class ConfigTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.client = await TestClient.get_client()
        self.user = await UserFactory.create()
        self.team_user = await TeamUserFactory.create()
        self.team_user_auth_client = await TestClient.get_logged_in_client(
            auth_user=self.team_user
        )

    def test_setup_script_without_logged_in_should_fail(self):
        resp = self.client.get("/api/v1/config/setup-script")
        assert resp.status_code == 401
        assert resp.json() == {"message": "Not authenticated"}

    def test_setup_script_should_pass(self):
        resp = self.team_user_auth_client.get("/api/v1/config/setup-script")
        assert resp.json() == {
            "message": f"portr auth set --token {self.team_user.secret_key} --remote {settings.server_url}"
        }

    def test_download_config_should_pass(self):
        resp = self.team_user_auth_client.post(
            "/api/v1/config/download", json={"secret_key": self.team_user.secret_key}
        )
        assert resp.json() == {
            "message": f"server_url: {settings.server_url}\nssh_url: {settings.ssh_url}\nsecret_key: {self.team_user.secret_key}\nenable_request_logging: false\nconnection_log_retention_days: 0\ntunnels:\n  - name: portr\n    subdomain: portr\n    port: 4321"
        }

    def test_download_config_with_wrong_secret_key_should_fail(self):
        resp = self.team_user_auth_client.post(
            "/api/v1/config/download", json={"secret_key": "random-secret-key"}
        )
        assert resp.json() == {"message": "Invalid secret key"}
