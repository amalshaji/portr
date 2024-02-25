import datetime
from pydantic import BaseModel

from portr_admin.schemas.user import TeamUserSchemaForConnection


class ConnectionSchema(BaseModel):
    id: str
    type: str
    subdomain: str | None
    port: int | None
    status: str
    created_at: datetime.datetime
    started_at: datetime.datetime | None
    closed_at: datetime.datetime | None

    created_by: TeamUserSchemaForConnection


class ConnectionCreateSchema(BaseModel):
    connection_type: str
    secret_key: str
    subdomain: str | None
