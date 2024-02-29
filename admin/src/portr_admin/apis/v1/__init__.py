from fastapi import APIRouter
from portr_admin.apis.v1.auth import api as api_v1_auth
from portr_admin.apis.v1.team import api as api_v1_team
from portr_admin.apis.v1.user import api as api_v1_user
from portr_admin.apis.v1.connection import api as api_v1_connection
from portr_admin.apis.v1.settings import api as api_v1_settings
from portr_admin.apis.v1.config import api as api_v1_config

api = APIRouter(prefix="/v1")
api.include_router(api_v1_auth)
api.include_router(api_v1_team)
api.include_router(api_v1_user)
api.include_router(api_v1_connection)
api.include_router(api_v1_settings)
api.include_router(api_v1_config)
