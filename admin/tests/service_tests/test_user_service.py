from unittest.mock import patch
import pytest
from tortoise.contrib import test
from models.user import GithubUser, User
from services import user as user_service
from tests.factories import TeamUserFactory, UserFactory


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
    async def test_get_or_create_user_creates_superuser(
        self, get_access_token_fn, get_user_fn
    ):
        await User.filter().delete()

        get_access_token_fn.return_value = "token"
        get_user_fn.return_value = {
            "id": 123,
            "email": "example@example.com",
            "avatar_url": "",
        }

        user = await user_service.get_or_create_user_from_github("code")

        assert str(user.email) == "example@example.com"
        assert user.is_superuser

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

    async def test_is_user_active_with_not_part_of_any_team(self):
        assert not await user_service.is_user_active(self.user)

    async def test_is_user_active_with_is_superuser(self):
        superuser = await UserFactory.create(
            email="example@example.com", is_superuser=True
        )

        assert await user_service.is_user_active(superuser)
