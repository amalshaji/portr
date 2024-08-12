from pydantic import BaseModel, EmailStr

from models.user import Role


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
