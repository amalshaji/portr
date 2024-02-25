#!/bin/bash

python scripts/pre-deploy.py
uvicorn src.portr_admin.main:app --host 0.0.0.0