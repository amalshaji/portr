import pytest
from utils.template_renderer import render_template


@pytest.fixture
def context():
    return {"name": "John", "age": 30}


@pytest.mark.parametrize(
    "test_input,expected",
    [
        (
            "Hello, my name is {{ name }} and I am {{ age }} years old.",
            "Hello, my name is John and I am 30 years old.",
        ),
        (
            "Hello, my name is {{ name }} and I am {{ age years old.",
            "Hello, my name is John and I am {{ age years old.",
        ),
    ],
)
def test_render_template(context, test_input, expected):
    assert render_template(test_input, context) == expected
