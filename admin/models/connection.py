from tortoise import Model, fields
from config.enums import Enum

from models import TimestampModelMixin

from models.user import Team, TeamUser
from utils.token import generate_connection_id


class ConnectionType(str, Enum):
    http = "http"
    tcp = "tcp"


class ConnectionStatus(str, Enum):
    reserved = "reserved"
    active = "active"
    closed = "closed"


class Connection(TimestampModelMixin, Model):
    id = fields.CharField(max_length=26, pk=True, default=generate_connection_id)
    type = fields.CharField(max_length=255, choices=ConnectionType.choices())
    subdomain = fields.CharField(max_length=255, null=True)
    port = fields.IntField(null=True)
    status = fields.CharField(
        max_length=255,
        choices=ConnectionStatus.choices(),
        default=ConnectionStatus.reserved.value,  # type: ignore
        index=True,
    )
    created_by: fields.ForeignKeyRelation[TeamUser] = fields.ForeignKeyField(
        "models.TeamUser", related_name="connections"
    )
    started_at = fields.DatetimeField(null=True)
    closed_at = fields.DatetimeField(null=True)
    team: fields.ForeignKeyRelation[Team] = fields.ForeignKeyField(
        "models.Team", related_name="connections"
    )
