from unittest.mock import patch

import pytest
from tortoise.contrib import test

from models.user import GithubUser, TeamUser, User
from services import user as user_service
from tests.factories import TeamUserFactory, UserFactory
from utils.exception import ServiceError


class TestUserService(test.TruncationTestCase):
    async def asyncSetUp(self) -> None:
        await super().asyncSetUp()
        self.user = await UserFactory.create(email="amal@portr.dev")

    @patch("services.user.GithubOauth.get_emails")
    @patch("services.user.GithubOauth.get_user")
    @patch("services.user.GithubOauth.get_access_token")
    async def test_get_or_create_user_from_github_with_remote_data(
        self, get_access_token_fn, get_user_fn, get_emails_fn
    ):
        get_access_token_fn.return_value = "token"
        get_user_fn.return_value = {"email": ""}
        get_emails_fn.return_value = []

        with pytest.raises(user_service.EmailFetchError) as e:
            await user_service.get_or_create_user_from_github("code")

        assert str(e.value) == "No verified email found"

    @patch("services.user.GithubOauth.get_user")
    @patch("services.user.GithubOauth.get_access_token")
    async def test_get_or_create_user_from_github_for_first_time(
        self, get_access_token_fn, get_user_fn
    ):
        await User.filter().delete()

        get_access_token_fn.return_value = "token"
        get_user_fn.return_value = {
            "id": 123,
            "email": "example@example.com",
            "avatar_url": "",
        }

        with pytest.raises(ServiceError) as e:
            await user_service.get_or_create_user_from_github("code")

        assert str(e.value) == "Password is required for the first user"

    @patch("services.user.GithubOauth.get_user")
    @patch("services.user.GithubOauth.get_access_token")
    async def test_get_or_create_user_superuser_from_github_with_not_part_of_any_team(
        self, get_access_token_fn, get_user_fn
    ):
        await UserFactory.create(email="example@example.com", is_superuser=True)

        get_access_token_fn.return_value = "token"
        get_user_fn.return_value = {
            "id": 123,
            "email": "example@example.com",
            "avatar_url": "",
        }

        user = await user_service.get_or_create_user_from_github("code")

        assert str(user.email) == "example@example.com"

    @patch("services.user.GithubOauth.get_user")
    @patch("services.user.GithubOauth.get_access_token")
    async def test_get_or_create_user_from_github_without_email(
        self, get_access_token_fn, get_user_fn
    ):
        get_access_token_fn.return_value = "token"
        get_user_fn.return_value = {"email": "example@example.com"}

        with pytest.raises(user_service.UserNotFoundError) as e:
            await user_service.get_or_create_user_from_github("code")

        assert str(e.value) == "User does not exist"

    @patch("services.user.GithubOauth.get_user")
    @patch("services.user.GithubOauth.get_access_token")
    async def test_get_or_create_user_from_github_with_existing_user(
        self, get_access_token_fn, get_user_fn
    ):
        await TeamUserFactory.create(user=self.user)

        get_access_token_fn.return_value = "token"
        get_user_fn.return_value = {
            "id": 123,
            "email": "amal@portr.dev",
            "avatar_url": "",
        }

        await user_service.get_or_create_user_from_github("code")

        assert await GithubUser.filter().count() == 1

        github_user = await GithubUser.filter().select_related("user").first()
        assert github_user.user == self.user
        assert github_user.github_id == 123
        assert github_user.github_access_token == "token"
        assert github_user.github_avatar_url == ""

    async def test_get_or_create_for_first_time(self):
        await self.user.delete()

        user = await user_service.get_or_create_user(
            email="amal@portr.dev", password="amal"
        )

        assert user.is_superuser is True
        assert user.email == "amal@portr.dev"
        assert user.check_password("amal")

        team = await user.teams.filter().first()
        assert team is not None
        assert team.name == "Portr"

        assert await TeamUser.filter(user=user, team=team).count() == 1

    async def test_get_or_create_with_non_existent_email(self):
        with pytest.raises(user_service.UserNotFoundError) as e:
            await user_service.get_or_create_user(
                email="amal@portr.com", password="amal"
            )

        assert str(e.value) == "User does not exist"

    async def test_get_or_create_with_wrong_password(self):
        with pytest.raises(user_service.WrongPasswordError) as e:
            await user_service.get_or_create_user(
                email="amal@portr.dev", password="amal"
            )

        assert str(e.value) == "Password is incorrect"
