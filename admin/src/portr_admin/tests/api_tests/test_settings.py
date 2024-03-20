from portr_admin.models.settings import GlobalSettings
from portr_admin.services.settings import (
    DEFAULT_ADD_USER_EMAIL_BODY,
    DEFAULT_ADD_USER_EMAIL_SUBJECT,
    populate_global_settings,
)
from portr_admin.tests import TestClient
from tortoise.contrib import test
from portr_admin.tests.factories import UserFactory


class SettingsTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.client = await TestClient.get_client()
        self.non_superuser = await UserFactory.create()
        self.superuser = await UserFactory.create(is_superuser=True)
        self.superuser_client = await TestClient.get_logged_in_client(self.superuser)
        self.non_superuser_client = await TestClient.get_logged_in_client(
            self.non_superuser
        )

        # move this to conftest.py
        await populate_global_settings()

    async def test_get_settings_with_no_login_should_fail(self):
        resp = self.client.get("/api/v1/settings/")
        assert resp.status_code == 401
        assert resp.json() == {"message": "Not authenticated"}

    async def test_get_settings_with_non_superuser_should_fail(self):
        resp = self.non_superuser_client.get("/api/v1/settings/")
        assert resp.status_code == 403
        assert resp.json() == {"message": "Only superuser can perform this action"}

    async def test_update_settings_with_no_login_should_fail(self):
        resp = self.client.patch("/api/v1/settings/")
        assert resp.status_code == 401
        assert resp.json() == {"message": "Not authenticated"}

    async def test_update_settings_with_non_superuser_should_fail(self):
        resp = self.non_superuser_client.patch("/api/v1/settings/")
        assert resp.status_code == 403
        assert resp.json() == {"message": "Only superuser can perform this action"}

    async def test_get_settings_with_superuser_should_pass(self):
        resp = self.superuser_client.get("/api/v1/settings/")
        assert resp.status_code == 200
        data = resp.json()

        assert data["smtp_enabled"] is False
        assert data["smtp_host"] is None
        assert data["smtp_port"] is None
        assert data["smtp_username"] is None
        assert data["from_address"] is None
        assert data["add_user_email_subject"] == DEFAULT_ADD_USER_EMAIL_SUBJECT
        assert data["add_user_email_body"] == DEFAULT_ADD_USER_EMAIL_BODY

    async def test_update_settings_with_superuser_should_pass(self):
        resp = self.superuser_client.patch(
            "/api/v1/settings/", json={"smtp_enabled": True}
        )
        assert resp.status_code == 200
        data = resp.json()

        assert data["smtp_enabled"] is True

        updated_settings = await GlobalSettings.first()
        assert updated_settings.smtp_enabled is True
