from typing import Any, Coroutine, Iterable
from tortoise import Model, fields
from tortoise.backends.base.client import BaseDBAsyncClient
from config.enums import Enum
import slugify  # type: ignore
from models import PkModelMixin, TimestampModelMixin
from utils.token import generate_secret_key
from passlib.hash import argon2  # type: ignore


class User(PkModelMixin, TimestampModelMixin, Model):  # type: ignore
    email = fields.CharField(max_length=255, unique=True)
    first_name = fields.CharField(max_length=255, null=True)
    last_name = fields.CharField(max_length=255, null=True)
    password = fields.CharField(max_length=255, null=True)
    is_superuser = fields.BooleanField(default=False)

    teams: fields.ManyToManyRelation["Team"]

    def set_password(self, password: str) -> None:
        self.password = argon2.hash(password)

    def check_password(self, password: str) -> bool:
        if self.password is None:
            return False
        return argon2.verify(password.encode(), self.password)


class GithubUser(PkModelMixin, Model):  # type: ignore
    github_id = fields.BigIntField(unique=True, index=True)
    github_access_token = fields.CharField(max_length=255)
    github_avatar_url = fields.CharField(max_length=255)
    user: fields.OneToOneRelation[User] = fields.OneToOneField(
        "models.User", related_name="github_user", on_delete=fields.CASCADE
    )


class Team(PkModelMixin, TimestampModelMixin, Model):  # type: ignore
    name = fields.CharField(max_length=255, unique=True)
    slug = fields.CharField(max_length=255, unique=True, index=True)
    users = fields.ManyToManyField(
        "models.User", related_name="teams", through="team_users"
    )

    async def _pre_save(  # type: ignore
        self,
        using_db: BaseDBAsyncClient | None = None,
        update_fields: Iterable[str] | None = None,
    ) -> Coroutine[Any, Any, None]:
        self.slug = slugify.slugify(self.name)
        return await super()._pre_save(using_db, update_fields)  # type: ignore


class Role(str, Enum):
    admin = "admin"
    member = "member"


class TeamUser(TimestampModelMixin, Model):
    user: fields.ForeignKeyRelation[User] = fields.ForeignKeyField(
        "models.User", related_name="team_users"
    )
    team: fields.ForeignKeyRelation[Team] = fields.ForeignKeyField(
        "models.Team", related_name="team_users"
    )
    secret_key = fields.CharField(
        max_length=42, unique=True, index=True, default=generate_secret_key
    )
    role = fields.CharField(
        max_length=255,
        choices=Role.choices(),
        default=Role.member.value,  # type: ignore
    )

    class Meta:
        table = "team_users"
