from tortoise import Model, fields

from models import PkModelMixin, TimestampModelMixin, EncryptedField
from models.user import User


class InstanceSettings(PkModelMixin, TimestampModelMixin, Model):  # type: ignore
    smtp_enabled = fields.BooleanField(default=False)
    smtp_host = fields.CharField(max_length=255, null=True)
    smtp_port = fields.IntField(null=True)
    smtp_username = fields.CharField(max_length=255, null=True)
    smtp_password = EncryptedField(max_length=255, null=True)
    from_address = fields.CharField(max_length=255, null=True)
    add_user_email_subject = fields.CharField(max_length=255, null=True)
    add_user_email_body = fields.TextField(null=True)

    updated_by: fields.ForeignKeyRelation[User] | None = fields.ForeignKeyField(
        "models.User",
        null=True,
        on_delete=fields.SET_NULL,
    )
