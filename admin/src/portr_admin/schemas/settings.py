import datetime
from pydantic import BaseModel

from portr_admin.schemas.user import UserSchema


class SettingsSchemaBase(BaseModel):
    updated_by: UserSchema | None
    updated_at: datetime.datetime


class SettingsUpdateSchema(BaseModel):
    smtp_enabled: bool
    smtp_host: str | None = None
    smtp_port: int | None = None
    smtp_username: str | None = None
    # smtp_password: str | None = None # We are not going to return the password
    from_address: str | None = None
    add_user_email_subject: str | None = None
    add_user_email_body: str | None = None


class SettingsResponseSchema(SettingsUpdateSchema, SettingsSchemaBase):
    pass
