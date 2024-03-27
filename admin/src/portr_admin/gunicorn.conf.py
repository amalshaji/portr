import os

bind = "0.0.0.0:8000"
workers = os.environ.get("GUNICORN_WORKERS", 2)
worker_class = "uvicorn.workers.UvicornWorker"
