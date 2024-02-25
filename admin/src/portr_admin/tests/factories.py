from portr_admin.models.auth import Session
from portr_admin.models.connection import Connection, ConnectionStatus, ConnectionType
from portr_admin.models.user import Team, TeamUser, User
from factory import SubFactory, Sequence, LazyAttribute  # type: ignore
from async_factory_boy.factory.tortoise import AsyncTortoiseFactory  # type: ignore
import mimesis


class UserFactory(AsyncTortoiseFactory):
    class Meta:
        model = User

    email = LazyAttribute(lambda _: mimesis.Person().email())


class SessionFactory(AsyncTortoiseFactory):
    class Meta:
        model = Session

    user = SubFactory(UserFactory)


class TeamFactory(AsyncTortoiseFactory):
    class Meta:
        model = Team

    name = Sequence(lambda n: f"test team-{n}")
    slug = Sequence(lambda n: f"test-team-{n}")


class TeamUserFactory(AsyncTortoiseFactory):
    class Meta:
        model = TeamUser

    user = SubFactory(UserFactory)
    team = SubFactory(TeamFactory)
    role = "admin"


class ConnectionFactory(AsyncTortoiseFactory):
    class Meta:
        model = Connection

    type = ConnectionType.http
    subdomain = LazyAttribute(lambda _: mimesis.Person().username())
    status = ConnectionStatus.reserved
    created_by = SubFactory(TeamUserFactory)
