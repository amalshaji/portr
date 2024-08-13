import datetime
from pydantic import BaseModel

from schemas.user import UserSchema


class SettingsSchemaBase(BaseModel):
    smtp_enabled: bool
    smtp_host: str | None = None
    smtp_port: int | None = None
    smtp_username: str | None = None
    from_address: str | None = None
    add_user_email_subject: str | None = None
    add_user_email_body: str | None = None


class SettingsUpdatedBySchema(BaseModel):
    updated_by: UserSchema | None
    updated_at: datetime.datetime


class SettingsUpdateSchema(SettingsSchemaBase):
    smtp_password: str | None = None


class SettingsResponseSchema(SettingsSchemaBase, SettingsUpdatedBySchema):
    pass
