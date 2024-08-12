from functools import cache
import json

from pathlib import Path

MANIFEST_PATH = (
    Path(__file__).parent.parent.parent / "web/dist/static/.vite/manifest.json"
)


@cache
def generate_vite_tags() -> str:
    if not MANIFEST_PATH.exists():
        raise FileNotFoundError("manifest.json not found")

    manifest_json = json.loads(MANIFEST_PATH.read_text())

    tag = ""

    for style in manifest_json["index.html"]["css"]:
        tag += f'<link rel="stylesheet" crossorigin href="/{style}">'

    if manifest_json["index.html"]["file"]:
        tag += f'<script type="module" crossorigin src="/{manifest_json["index.html"]["file"]}"></script>'

    return tag.strip()
