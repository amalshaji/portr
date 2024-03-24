from portr_admin.models.settings import InstanceSettings
import logging


DEFAULT_SMTP_ENABLED = False
DEFAULT_ADD_USER_EMAIL_SUBJECT = """
You've been added to team {{teamName}} on Portr!
""".strip()
DEFAULT_ADD_USER_EMAIL_BODY = """
Hello {{email}}

You've been added to team "{{teamName}}" on Portr.

Get started by signing in with your github account at {{dashboardUrl}}
""".strip()


async def populate_instance_settings():
    logger = logging.getLogger()
    settings = await InstanceSettings.first()
    if not settings:
        logger.info("Creating default instance settings")
        settings = await InstanceSettings.create(
            smtp_enabled=DEFAULT_SMTP_ENABLED,
            add_user_email_subject=DEFAULT_ADD_USER_EMAIL_SUBJECT,
            add_user_email_body=DEFAULT_ADD_USER_EMAIL_BODY,
        )
    return settings


async def get_instance_settings() -> InstanceSettings:
    settings = await InstanceSettings.filter().select_related("updated_by").first()
    if not settings:
        raise Exception("Instance settings not found")
    return settings
