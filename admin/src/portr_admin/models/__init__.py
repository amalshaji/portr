from typing import Any, Callable, List
from tortoise import Model, fields
from cryptography.fernet import Fernet
from tortoise.validators import Validator
from portr_admin.config import settings


class PkModelMixin(Model):
    id = fields.IntField(pk=True)

    class Meta:
        abstract = True


class TimestampModelMixin(Model):
    created_at = fields.DatetimeField(auto_now_add=True)
    updated_at = fields.DatetimeField(auto_now=True)

    class Meta:
        abstract = True


class EncryptedField(fields.BinaryField):  # type: ignore
    def __init__(
        self,
        source_field: str | None = None,
        generated: bool = False,
        pk: bool = False,
        null: bool = False,
        default: Any = None,
        unique: bool = False,
        index: bool = False,
        description: str | None = None,
        model: Model | None = None,
        validators: List[Validator | Callable[..., Any]] | None = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(
            source_field,
            generated,
            pk,
            null,
            default,
            unique,
            index,
            description,
            model,
            validators,
            **kwargs,
        )
        self.fernet = Fernet(settings.encryption_key)

    def to_db_value(self, value: Any, instance: Model | Model) -> Any:  # type: ignore
        if value is None:
            return None
        return self.fernet.encrypt(value.encode())

    def to_python_value(self, value: Any) -> Any:
        if value is None:
            return None
        return self.fernet.decrypt(value).decode()
