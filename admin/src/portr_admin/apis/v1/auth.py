import hashlib
import hmac
from fastapi import APIRouter, Depends, Header, Request, Response
from fastapi.responses import RedirectResponse
from portr_admin.config import settings
from portr_admin.models.user import User
from portr_admin.utils.github_auth import GithubOauth
from portr_admin.utils.token import generate_oauth_state
from portr_admin.services import user as user_service
from portr_admin.services import auth as auth_service
from portr_admin.apis import security
import logging
from fastapi import BackgroundTasks
import urllib.parse

api = APIRouter(prefix="/auth", tags=["auth"])

GITHUB_CALLBACK_URL = "/api/v1/auth/github/callback"


@api.get("/is-first-signup")
async def is_first_signup():
    return {"is_first_signup": await User.filter().count() == 0}


@api.get("/github")
async def github_login(request: Request):
    state = generate_oauth_state()
    redirect_uri = f"{settings.domain_address()}{GITHUB_CALLBACK_URL}?state={state}"

    client = GithubOauth(
        client_id=settings.github_app_client_id,
        client_secret=settings.github_app_client_secret,
    )

    response = RedirectResponse(url=client.auth_url(state, redirect_uri))
    response.set_cookie(
        key="oauth_state",
        value=state,
        httponly=True,
        max_age=600,
        secure=not settings.debug,
    )

    next_url = request.query_params.get("next")
    if next_url:
        response.set_cookie(
            key="portr_next_url",
            value=next_url,
            httponly=True,
            max_age=600,
            secure=not settings.debug,
        )

    return response


@api.get("/github/callback")
async def github_callback(request: Request, code: str, state: str):
    existing_state = request.cookies.get("oauth_state")
    if state != existing_state:
        return Response(status_code=400, content="Invalid state")

    user = await user_service.get_or_create_user_from_github(code)
    token = await auth_service.login_user(user)

    next_url_encoded = request.cookies.get("portr_next_url")
    next_url = urllib.parse.unquote(next_url_encoded) if next_url_encoded else None

    response = RedirectResponse(url=next_url or "/")
    response.set_cookie(
        key="portr_session",
        value=token,
        httponly=True,
        max_age=60 * 60 * 24 * 7,
        secure=not settings.debug,
    )
    response.delete_cookie(key="portr_next_url")

    return response


@api.get("/github/events")
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


@api.post("/logout")
async def logout(_=Depends(security.get_current_user)):
    response = Response()
    response.delete_cookie(key="portr_session")
    return response
