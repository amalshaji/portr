from fastapi import APIRouter, Depends, Request, Response
from fastapi.responses import JSONResponse, RedirectResponse
from config import settings
from models.user import User
from schemas.auth import LoginSchema
from utils.github_auth import GithubOauth
from utils.token import generate_oauth_state
from services import user as user_service
from services import auth as auth_service
from apis import security
import urllib.parse

api = APIRouter(prefix="/auth", tags=["auth"])

GITHUB_CALLBACK_URL = "/api/v1/auth/github/callback"


async def login_user(response: Response, user: User):
    token = await auth_service.login_user(user)
    response.set_cookie(
        key="portr_session",
        value=token,
        httponly=True,
        max_age=60 * 60 * 24 * 7,
        secure=not settings.debug,
    )
    response.delete_cookie(key="portr_next_url")
    return response


@api.get("/auth-config")
async def auth_config():
    return {
        "is_first_signup": await User.filter().count() == 0,
        "github_auth_enabled": settings.github_client_id is not None,
    }


@api.post("/login")
async def login(data: LoginSchema):
    try:
        user = await user_service.get_or_create_user(
            email=data.email, password=data.password
        )
    except user_service.UserNotFoundError as e:
        return JSONResponse(content={"email": str(e)}, status_code=400)
    except user_service.WrongPasswordError as e:
        return JSONResponse(content={"password": str(e)}, status_code=400)

    first_team = await user.teams.filter().first()
    response = JSONResponse(content={"redirect_to": f"/{first_team.slug}/overview"})

    return await login_user(response, user)


@api.get("/github")
async def github_login(request: Request):
    state = generate_oauth_state()
    redirect_uri = f"{settings.domain_address()}{GITHUB_CALLBACK_URL}"
    client = GithubOauth(
        client_id=settings.github_client_id,
        client_secret=settings.github_client_secret,
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
        return RedirectResponse(url="/?code=invalid-state")

    try:
        user = await user_service.get_or_create_user_from_github(code)
    except user_service.UserNotFoundError:
        return RedirectResponse(url="/?code=user-not-found")
    except user_service.EmailFetchError:
        return RedirectResponse(url="/?code=private-email")

    next_url_encoded = request.cookies.get("portr_next_url")
    next_url = urllib.parse.unquote(next_url_encoded) if next_url_encoded else None

    response = RedirectResponse(url=next_url or "/")

    return await login_user(response, user)


@api.post("/logout")
async def logout(_=Depends(security.get_current_user)):
    response = Response()
    response.delete_cookie(key="portr_session")
    return response
