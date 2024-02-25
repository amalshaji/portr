from unittest.mock import MagicMock
from portr_admin.models.connection import ConnectionType
import pytest
from tortoise.contrib.test import SimpleTestCase
from portr_admin.services import connection as connection_service
from portr_admin.utils.exception import ServiceError
from unittest.mock import patch


class ConnectionServiceTests(SimpleTestCase):
    def setUp(self) -> None:
        super().setUp()
        self.team_user = MagicMock(team=MagicMock())

    async def test_create_http_connection_without_subdomain_should_fail(self):
        with pytest.raises(ServiceError) as e:
            await connection_service.create_new_connection(
                type=ConnectionType.http, created_by=MagicMock(), subdomain=None
            )
        assert str(e.value) == "subdomain is required for http connections"

    @patch("portr_admin.models.connection.Connection.create")
    async def test_create_http_connection_with_subdomain_should_succeed(
        self, create_fn
    ):
        await connection_service.create_new_connection(
            type=ConnectionType.http,
            created_by=self.team_user,
            subdomain="test-subdomain",
        )

        create_fn.assert_called_once_with(
            type=ConnectionType.http,
            subdomain="test-subdomain",
            port=None,
            created_by=self.team_user,
            team=self.team_user.team,
        )

    @patch("portr_admin.models.connection.Connection.create")
    async def test_create_tcp_connection_should_succeed(self, create_fn):
        await connection_service.create_new_connection(
            type=ConnectionType.tcp,
            created_by=self.team_user,
            subdomain="test-subdomain",
        )

        create_fn.assert_called_once_with(
            type=ConnectionType.tcp,
            subdomain=None,
            port=None,
            created_by=self.team_user,
            team=self.team_user.team,
        )
