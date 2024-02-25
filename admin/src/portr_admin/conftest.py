import os
import pytest
from tortoise.contrib.test import finalizer, initializer

from portr_admin.db import TORTOISE_MODELS


@pytest.fixture(scope="session", autouse=True)
def initialize_tests(request):
    db_url = os.environ.get("TORTOISE_TEST_DB", "sqlite://:memory:")
    initializer(TORTOISE_MODELS, db_url=db_url, app_label="models")
    request.addfinalizer(finalizer)
