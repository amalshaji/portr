import datetime
from pydantic import BaseModel, EmailStr

from portr_admin.models.user import Role
from portr_admin.schemas.user import TeamUserSchemaForConnection


class NewTeamSchema(BaseModel):
    name: str


class TeamSchema(BaseModel):
    id: int
    name: str
    slug: str


class AddUserToTeamSchema(BaseModel):
    email: EmailStr
    role: Role
    set_superuser: bool


class TeamSettingsSchemaBase(BaseModel):
    updated_by: TeamUserSchemaForConnection | None
    updated_at: datetime.datetime


class TeamSettingsUpdateSchema(BaseModel):
    github_org_webhook_secret: str | None = None
    github_org_pat: str | None = None
    auto_invite_github_org_members: bool = False


class TeamSettingsResponseSchema(TeamSettingsSchemaBase, TeamSettingsUpdateSchema):
    pass
