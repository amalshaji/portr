#!/bin/sh

python -c "import base64, os; print(base64.urlsafe_b64encode(os.urandom(32)).decode())"