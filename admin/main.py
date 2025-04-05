from datetime import datetime, UTC
import os
from typing import Annotated
from fastapi import Cookie, FastAPI, Request
from fastapi.responses import JSONResponse, RedirectResponse
from apis import api as api_v1
from apscheduler.schedulers.asyncio import AsyncIOScheduler  # type: ignore
from apis.security import NotAuthenticated, get_current_user
from config.beats import clear_expired_sessions, clear_unclaimed_connections
from config import settings
from config.database import connect_db, disconnect_db
from models.user import User
from utils.exception import PermissionDenied, ServiceError
from fastapi.templating import Jinja2Templates

from utils.vite import generate_vite_tags
import urllib.parse
from fastapi import status
from contextlib import asynccontextmanager
from fastapi.staticfiles import StaticFiles

templates = Jinja2Templates(directory="templates")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # connect to database
    await connect_db()
    app.state.server_start_time = datetime.now(tz=UTC)
    yield
    # disconnect all db connections
    await disconnect_db()


app = FastAPI(lifespan=lifespan)
app.include_router(api_v1)


scheduler = AsyncIOScheduler()
scheduler.add_job(clear_expired_sessions, "interval", hours=1)
scheduler.add_job(clear_unclaimed_connections, "interval", seconds=10)
scheduler.start()


@app.get("/")
async def render_index_template(
    request: Request,
    portr_session: Annotated[str | None, Cookie()] = None,
):
    try:
        user: User = await get_current_user(portr_session)
    except NotAuthenticated:
        return templates.TemplateResponse(
            request=request,
            name="index.html",
            context={
                "request": request,
                "use_vite": settings.use_vite,
                "vite_tags": "" if settings.use_vite else generate_vite_tags(),
            },
        )

    first_team = await user.teams.filter().first()
    if first_team is None:
        return RedirectResponse(url="/instance-settings/team")

    return RedirectResponse(url=f"/{first_team.slug}/overview")


@app.get("/instance-settings/{rest:path}")
async def render_index_template_for_instance_settings_routes(
    request: Request,
    portr_session: Annotated[str | None, Cookie()] = None,
):
    try:
        user: User = await get_current_user(portr_session)
    except NotAuthenticated:
        next_url = request.url.path + "?" + request.url.query
        next_url_encoded = urllib.parse.urlencode({"next": next_url})
        return RedirectResponse(url=f"/?{next_url_encoded}")

    if not user.is_superuser:
        return RedirectResponse(url="/")

    return templates.TemplateResponse(
        request=request,
        name="index.html",
        context={
            "request": request,
            "use_vite": settings.use_vite,
            "vite_tags": "" if settings.use_vite else generate_vite_tags(),
        },
    )


@app.get("/{team}/overview")
@app.get("/{team}/connections")
@app.get("/{team}/users")
@app.get("/{team}/my-account")
@app.get("/{team}/email-settings")
async def render_index_template_for_team_routes(
    request: Request,
    team: str,
    portr_session: Annotated[str | None, Cookie()] = None,
):
    try:
        user: User = await get_current_user(portr_session)
    except NotAuthenticated:
        next_url = request.url.path + "?" + request.url.query
        next_url_encoded = urllib.parse.urlencode({"next": next_url})
        return RedirectResponse(url=f"/?{next_url_encoded}")

    team = await user.teams.filter(slug=team).first()  # type: ignore
    if team is None:
        return RedirectResponse(url="/")

    return templates.TemplateResponse(
        request=request,
        name="index.html",
        context={
            "request": request,
            "use_vite": settings.use_vite,
            "vite_tags": "" if settings.use_vite else generate_vite_tags(),
        },
    )


@app.exception_handler(NotAuthenticated)
async def not_authenticated_exception_handler(
    request: Request, exception: NotAuthenticated
):
    return JSONResponse(
        status_code=status.HTTP_401_UNAUTHORIZED,
        content={"message": "Not authenticated"},
    )


@app.exception_handler(ServiceError)
async def service_error_exception_handler(request: Request, exception: ServiceError):
    return JSONResponse(
        status_code=status.HTTP_400_BAD_REQUEST, content={"message": exception.message}
    )


@app.exception_handler(PermissionDenied)
async def permission_denied_exception_handler(
    request: Request, exception: PermissionDenied
):
    return JSONResponse(
        status_code=status.HTTP_403_FORBIDDEN, content={"message": exception.message}
    )


app.mount("/static", StaticFiles(directory="static"), name="static")
if not settings.use_vite:
    app.mount("/", StaticFiles(directory="web/dist/static"), name="web-static")


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8000,
        workers=int(os.environ.get("UVICORN_WORKERS", 2)),
        log_level="info",
    )
