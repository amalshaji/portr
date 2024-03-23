from typing import TypedDict
import httpx


class GithubUser(TypedDict):
    id: int
    email: str
    avatar_url: str


class GithubUserEmail(TypedDict):
    email: str
    verified: bool
    primary: bool
    visibility: str


class GithubOauth:
    AUTH_ENDPOINT = "https://github.com/login/oauth/authorize"
    TOKEN_ENDPOINT = "https://github.com/login/oauth/access_token"
    USER_ENDPOINT = "https://api.github.com/user"
    EMAILS_ENDPOINT = "https://api.github.com/user/emails"

    def __init__(self, client_id, client_secret):
        self.client_id = client_id
        self.client_secret = client_secret

    def auth_url(self, state: str, redirect_uri: str):
        return f"https://github.com/login/oauth/authorize?client_id={self.client_id}&redirect_uri={redirect_uri}&state={state}&scope=user:email"

    async def get_access_token(self, code: str) -> str:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                self.TOKEN_ENDPOINT,
                data={
                    "client_id": self.client_id,
                    "client_secret": self.client_secret,
                    "code": code,
                },
                headers={"Accept": "application/json"},
            )
            response.raise_for_status()
            return response.json()["access_token"]

    async def get_user(self, access_token: str) -> GithubUser:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                self.USER_ENDPOINT,
                headers={
                    "Authorization": f"Bearer {access_token}",
                    "Accept": "application/json",
                },
            )
            response.raise_for_status()
            return response.json()

    async def get_emails(self, access_token: str) -> list[GithubUserEmail]:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                self.EMAILS_ENDPOINT,
                headers={
                    "Authorization": f"Bearer {access_token}",
                    "Accept": "application/json",
                },
            )
            response.raise_for_status()
            return response.json()
