import hashlib
import hmac
import logging
from fastapi import APIRouter, BackgroundTasks, Header, Request, Response
from portr_admin.services import auth as auth_service
from portr_admin.config import settings

api = APIRouter(prefix="/org_events", tags=["org_events"])


@api.post("/")
async def github_webhook_events(
    request: Request,
    background_tasks: BackgroundTasks,
    x_hub_signature_256: str = Header(alias="X-Hub-Signature-256"),
):
    body = await request.body()
    hash_object = hmac.new(
        settings.github_webhook_secret.encode("utf-8"),
        msg=body,
        digestmod=hashlib.sha256,
    )
    expected_signature = "sha256=" + hash_object.hexdigest()
    if hmac.compare_digest(expected_signature, x_hub_signature_256):
        background_tasks.add_task(
            auth_service.process_github_webhook, body.decode("utf-8")
        )
        return Response(status_code=200)

    logger = logging.getLogger()
    logger.error("Failed to validate webhook origin, invalid signature")
    return Response(status_code=400)
