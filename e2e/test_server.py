import requests  # type: ignore


def test_json_api():
    response = requests.get("http://test-tunnel-server.localhost:8001")
    assert response.status_code == 200
    assert response.json() == {"Hello": "World"}
