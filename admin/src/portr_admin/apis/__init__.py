from fastapi import APIRouter
from portr_admin.apis.v1 import api as api_v1

api = APIRouter(prefix="/api")
api.include_router(api_v1)
