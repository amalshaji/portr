from typing import Generic, TypeVar
from pydantic import BaseModel, ConfigDict
from tortoise.queryset import QuerySet
from tortoise.models import Model

T = TypeVar("T")
Qs_T = TypeVar("Qs_T", bound=Model)


class PaginatedResponse(BaseModel, Generic[T]):
    count: int
    data: list[T]

    model_config = ConfigDict(arbitrary_types_allowed=True)

    @classmethod
    async def generate_response_for_page(
        self,
        qs: QuerySet[Qs_T],
        page: int,
        page_size: int = 10,
    ):
        if page < 1:
            page = 1

        self.count = await qs.count()
        self.data = await qs.limit(page_size).offset((page - 1) * page_size)
        return PaginatedResponse[T](count=self.count, data=self.data)  # type: ignore
