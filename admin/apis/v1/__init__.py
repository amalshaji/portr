from fastapi import APIRouter
from apis.v1.auth import api as api_v1_auth
from apis.v1.team import api as api_v1_team
from apis.v1.user import api as api_v1_user
from apis.v1.connection import api as api_v1_connection
from apis.v1.instance_settings import api as api_v1_instance_settings
from apis.v1.config import api as api_v1_config

api = APIRouter(prefix="/v1")
api.include_router(api_v1_auth)
api.include_router(api_v1_team)
api.include_router(api_v1_user)
api.include_router(api_v1_connection)
api.include_router(api_v1_instance_settings)
api.include_router(api_v1_config)


@api.get("/healthcheck", tags=["healthcheck"])
async def healthcheck():
    return {"status": "ok"}
