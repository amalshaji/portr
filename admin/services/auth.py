from models.auth import Session
from models.user import User


async def login_user(user: User) -> str:
    session = await Session.create(user=user)
    return session.token
