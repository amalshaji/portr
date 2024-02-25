from enum import Enum as BaseEnum
from typing import Any


class Enum(BaseEnum):
    @classmethod
    def choices(self) -> list[tuple[str, Any]]:
        return [(e.name, e.value) for e in self]
