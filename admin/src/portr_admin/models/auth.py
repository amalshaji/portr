import datetime
from tortoise import Model, fields

from portr_admin.models import PkModelMixin, TimestampModelMixin
from portr_admin.models.user import User
from portr_admin.utils.token import generate_session_token


class Session(PkModelMixin, TimestampModelMixin, Model):  # type: ignore
    user: fields.ForeignKeyRelation[User] = fields.ForeignKeyField(
        "models.User", related_name="sessions"
    )
    token = fields.CharField(
        max_length=255, unique=True, default=generate_session_token
    )
    expires_at = fields.DatetimeField(
        index=True,
        default=lambda: datetime.datetime.now(datetime.UTC)
        + datetime.timedelta(days=7),
    )
