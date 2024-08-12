from fastapi import APIRouter, Depends
from apis import security

from models.user import Team, TeamUser, User
from schemas.team import TeamSchema
from schemas.user import (
    ChangePasswordSchema,
    TeamUserSchemaForCurrentUser,
    UserSchema,
    UserUpdateSchema,
)
from utils.token import generate_secret_key

api = APIRouter(prefix="/user", tags=["user"])


@api.get("/me", response_model=TeamUserSchemaForCurrentUser)
async def current_team_user(
    team_user: TeamUser = Depends(security.get_current_team_user),
):
    return team_user


@api.get("/me/teams", response_model=list[TeamSchema])
async def current_user_teams(
    user: TeamUser = Depends(security.get_current_user),
):
    return await Team.filter(team_users__user=user).all()


@api.patch("/me/update", response_model=UserSchema)
async def update_user(
    data: UserUpdateSchema, user: User = Depends(security.get_current_user)
):
    for k, v in data.model_dump().items():
        if v is not None:
            setattr(user, k, v)
    await user.save()
    return user


@api.patch("/me/change_password", response_model=UserSchema)
async def change_password(
    data: ChangePasswordSchema, user: User = Depends(security.get_current_user)
):
    user.set_password(data.password)
    await user.save()
    return user


@api.patch("/me/rotate-secret-key")
async def rotate_secret_key(user: TeamUser = Depends(security.get_current_team_user)):
    user.secret_key = generate_secret_key()
    await user.save()
    return {"secret_key": user.secret_key}
