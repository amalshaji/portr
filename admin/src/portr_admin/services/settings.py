from portr_admin.models.settings import GlobalSettings
import logging


DEFAULT_SMTP_ENABLED = False
DEFAULT_ADD_USER_EMAIL_SUBJECT = """
You've been added to team {{teamName}} on Portr!
""".strip()
DEFAULT_ADD_USER_EMAIL_BODY = """
Hello {{email}}

You've been added to team "{{teamName}}" on Portr.

Get started by signing in with your github account at {{appUrl}}
""".strip()


async def populate_global_settings():
    logger = logging.getLogger()
    settings = await GlobalSettings.first()
    if not settings:
        logger.info("Creating default global settings")
        settings = await GlobalSettings.create(
            smtp_enabled=DEFAULT_SMTP_ENABLED,
            add_user_email_subject=DEFAULT_ADD_USER_EMAIL_SUBJECT,
            add_user_email_body=DEFAULT_ADD_USER_EMAIL_BODY,
        )
    return settings


async def get_global_settings() -> GlobalSettings:
    settings = await GlobalSettings.first()
    if not settings:
        raise Exception("Global settings not found")
    return settings
