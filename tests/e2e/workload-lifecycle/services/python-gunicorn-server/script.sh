#!/bin/bash

# export ENV_IN_SCRIPT=ENV_IN_SCRIPT_VALUE

# echo "ENV_IN_SCRIPT is: $ENV_IN_SCRIPT"
python3 script.py

exec  /app/venv/bin/gunicorn -w 5 -k uvicorn.workers.UvicornWorker main:app -b 0.0.0.0:8000


# /app/venv/bin/newrelic-admin run-program