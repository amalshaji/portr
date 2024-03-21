from fastapi import APIRouter, Depends
from portr_admin.apis import security
from portr_admin.models.user import User
from portr_admin.services import settings as settings_service
from portr_admin.schemas.settings import SettingsResponseSchema, SettingsUpdateSchema

api = APIRouter(prefix="/instance-settings", tags=["instance-settings"])


@api.get("/", response_model=SettingsResponseSchema)
async def get_settings(_=Depends(security.requires_superuser)):
    return await settings_service.get_instance_settings()


@api.patch("/", response_model=SettingsResponseSchema)
async def update_settings(
    data: SettingsUpdateSchema, user: User = Depends(security.requires_superuser)
):
    settings = await settings_service.get_instance_settings()

    if data.smtp_enabled is False:
        settings.smtp_enabled = False
        await settings.save()
        return settings

    for k, v in data.model_dump().items():
        setattr(settings, k, v)

    settings.updated_by = user  # type: ignore

    await settings.save()

    return settings
