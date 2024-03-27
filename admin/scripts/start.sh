#!/bin/bash

python scripts/pre-deploy.py
gunicorn --config src/portr_admin/gunicorn.conf.py src.portr_admin.main:app