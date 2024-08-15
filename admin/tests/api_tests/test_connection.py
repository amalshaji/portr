from models.connection import Connection, ConnectionStatus
from tests import TestClient
from tortoise.contrib import test

from tests.factories import ConnectionFactory, TeamUserFactory


class ConnectionTests(test.TestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.team_user = await TeamUserFactory.create()
        self.client = await TestClient.get_client()
        self.team_user_client = await TestClient.get_logged_in_client(self.team_user)

        self.active_connection_1 = await ConnectionFactory.create(
            status=ConnectionStatus.active, team=self.team_user.team
        )
        self.closed_connection_2 = await ConnectionFactory.create(
            status=ConnectionStatus.closed, team=self.team_user.team
        )

    async def test_create_new_connection(self):
        resp = self.client.post(
            "/api/v1/connections/",
            json={
                "connection_type": "http",
                "secret_key": self.team_user.secret_key,
                "subdomain": "test-subdomain",
            },
        )
        assert resp.status_code == 200
        created_connection_id = resp.json()["connection_id"]

        created_connection = (
            await Connection.filter(id=created_connection_id)
            .select_related("created_by")
            .first()
        )

        assert created_connection is not None
        assert created_connection.created_by == self.team_user
        assert created_connection.subdomain == "test-subdomain"
        assert created_connection.port is None
        assert created_connection.type == "http"
        assert created_connection.status == "reserved"

    async def test_create_new_connection_with_wrong_secret_key_should_fail(self):
        resp = self.client.post(
            "/api/v1/connections/",
            json={
                "connection_type": "http",
                "secret_key": "random-secret-key",
                "subdomain": "test-subdomain",
            },
        )
        assert resp.status_code == 400
        assert resp.json() == {"message": "Invalid secret key"}

    async def test_create_new_connection_with_active_subdomain_should_fail(self):
        await ConnectionFactory.create(
            type="http",
            subdomain="test-subdomain",
            team=self.team_user.team,
            status=ConnectionStatus.active,
        )
        resp = self.client.post(
            "/api/v1/connections/",
            json={
                "connection_type": "http",
                "secret_key": self.team_user.secret_key,
                "subdomain": "test-subdomain",
            },
        )
        assert resp.status_code == 400
        assert resp.json() == {"message": "Subdomain already in use"}

    async def test_list_active_connections(self):
        resp = self.team_user_client.get(
            "/api/v1/connections/",
            params={"type": "active"},
        )
        assert resp.status_code == 200
        assert resp.json()["count"] == 1
        assert resp.json()["data"][0]["id"] == self.active_connection_1.id

    async def test_list_recent_connections(self):
        resp = self.team_user_client.get(
            "/api/v1/connections/",
            params={"type": "recent"},
        )
        assert resp.status_code == 200
        assert resp.json()["count"] == 2

    async def test_list_recent_connections_pagination(self):
        resp = self.team_user_client.get(
            "/api/v1/connections/?page_size=1",
            params={"type": "recent"},
        )
        assert resp.status_code == 200
        assert resp.json()["count"] == 2
        assert len(resp.json()["data"]) == 1
