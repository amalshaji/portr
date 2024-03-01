import aiosmtplib
from portr_admin.models.settings import GlobalSettings
from email.message import EmailMessage


async def send_mail(to: str, subject: str, body: str):
    settings = await GlobalSettings.first()

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
