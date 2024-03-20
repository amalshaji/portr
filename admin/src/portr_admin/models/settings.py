from tortoise import Model, fields

from portr_admin.models import PkModelMixin, TimestampModelMixin, EncryptedField


class GlobalSettings(PkModelMixin, TimestampModelMixin, Model):  # type: ignore
    smtp_enabled = fields.BooleanField(default=False)
    smtp_host = fields.CharField(max_length=255, null=True)
    smtp_port = fields.IntField(null=True)
    smtp_username = fields.CharField(max_length=255, null=True)
    smtp_password = EncryptedField(max_length=255, null=True)
    from_address = fields.CharField(max_length=255, null=True)
    add_user_email_subject = fields.CharField(max_length=255, null=True)
    add_user_email_body = fields.TextField(null=True)
