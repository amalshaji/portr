class PortrError(Exception):
    def __init__(self, message: str | None = None) -> None:
        self.message = message


class ServiceError(PortrError):
    pass


class PermissionDenied(PortrError):
    pass
