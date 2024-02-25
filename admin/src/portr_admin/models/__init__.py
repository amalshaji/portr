from tortoise import Model, fields


class PkModelMixin(Model):
    id = fields.IntField(pk=True)

    class Meta:
        abstract = True


class TimestampModelMixin(Model):
    created_at = fields.DatetimeField(auto_now_add=True)
    updated_at = fields.DatetimeField(auto_now=True)

    class Meta:
        abstract = True
