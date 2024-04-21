from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    debug: bool = Field(default=False, alias="PORTR_ADMIN_DEBUG")
    db_url: str = "sqlite://db.sqlite"
    domain: str
    use_vite: bool = Field(default=False, alias="PORTR_ADMIN_USE_VITE")
    encryption_key: str = Field(alias="PORTR_ADMIN_ENCRYPTION_KEY")

    github_client_id: str = Field(alias="PORTR_ADMIN_GITHUB_CLIENT_ID", default="")
    github_client_secret: str = Field(alias="PORTR_ADMIN_GITHUB_CLIENT_SECRET", default="")

    remote_user_header: str = Field(alias="PORTR_ADMIN_REMOTE_USER_HEADER", default="")

    server_url: str
    ssh_url: str

    model_config = SettingsConfigDict(env_file=".env", env_prefix="PORTR_")

    def domain_address(self):
        if "localhost:" in self.domain:
            return f"http://{self.domain}"
        return f"https://{self.domain}"


settings = Settings()  # type: ignore
