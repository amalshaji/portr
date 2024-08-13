import aiosmtplib
from email.message import EmailMessage
from services import settings as settings_service


async def send_mail(to: str, subject: str, body: str):
    settings = await settings_service.get_instance_settings()

    message = EmailMessage()
    message["From"] = settings.from_address  # type: ignore
    message["To"] = to
    message["Subject"] = subject
    message.set_content(body)

    await aiosmtplib.send(
        message,
        hostname=settings.smtp_host,  # type: ignore
        port=settings.smtp_port,  # type: ignore
        username=settings.smtp_username,  # type: ignore
        password=settings.smtp_password,  # type: ignore
        use_tls=True,
    )
