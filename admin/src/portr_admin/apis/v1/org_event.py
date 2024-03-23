import hashlib
import hmac
import logging
from fastapi import APIRouter, BackgroundTasks, Header, Request, Response
from portr_admin.models.user import TeamSettings
from portr_admin.services import auth as auth_service

api = APIRouter(prefix="/org_events", tags=["org_events"])


@api.post("/")
async def github_webhook_events(
    request: Request,
    background_tasks: BackgroundTasks,
    team: str,
    x_hub_signature_256: str = Header(alias="X-Hub-Signature-256"),
):
    team_settings = (
        await TeamSettings.filter().select_related("team").get_or_none(team__slug=team)
    )
    if not team_settings:
        return Response(status_code=401)

    body = await request.body()
    hash_object = hmac.new(
        team_settings.github_org_webhook_secret.encode("utf-8"),  # type: ignore
        msg=body,
        digestmod=hashlib.sha256,
    )
    expected_signature = "sha256=" + hash_object.hexdigest()
    if hmac.compare_digest(expected_signature, x_hub_signature_256):
        background_tasks.add_task(
            auth_service.process_github_webhook,
            team_settings.team,
            body.decode("utf-8"),
        )
        return Response(status_code=200)

    logger = logging.getLogger()
    logger.error("Failed to validate webhook origin, invalid signature")
    return Response(status_code=400)
