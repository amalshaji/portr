from fastapi import APIRouter, Depends
from portr_admin.apis import security
from portr_admin.models.settings import GlobalSettings

from portr_admin.schemas.settings import SettingsSchema

api = APIRouter(prefix="/settings", tags=["settings"])


@api.get("/", response_model=SettingsSchema)
async def get_settings(_=Depends(security.requires_superuser)):
    settings = await GlobalSettings.first()
    if not settings:
        raise Exception("Global settings not found")

    return settings


@api.patch("/", response_model=SettingsSchema)
async def update_settings(data: SettingsSchema, _=Depends(security.requires_superuser)):
    settings = await GlobalSettings.first()
    if not settings:
        raise Exception("Global settings not found")

    if data.smtp_enabled is False:
        settings.smtp_enabled = False
        await settings.save()
        return settings

    for k, v in data.model_dump().items():
        setattr(settings, k, v)
    await settings.save()

    return settings
