import string
import nanoid  # type: ignore
from ulid import ULID

NANOID_ALPHABETS = string.ascii_letters + string.digits


def generate_secret_key() -> str:
    return f"portr_{nanoid.generate(size=36, alphabet=NANOID_ALPHABETS)}"


def generate_oauth_state() -> str:
    return nanoid.generate(size=26)


def generate_session_token() -> str:
    return nanoid.generate(size=32)


def generate_connection_id() -> str:
    return str(ULID())


def generate_random_password() -> str:
    return nanoid.generate(size=16)
