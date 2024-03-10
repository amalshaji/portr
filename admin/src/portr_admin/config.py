from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    debug: bool = False
    db_url: str = "sqlite://db.sqlite"
    domain: str
    use_vite: bool = False

    github_app_client_id: str
    github_app_client_secret: str

    server_url: str
    ssh_url: str

    model_config = SettingsConfigDict(env_file=".env")

    def domain_address(self):
        if "localhost:" in self.domain:
            return f"http://{self.domain}"
        return f"https://{self.domain}"


settings = Settings()  # type: ignore
