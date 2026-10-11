#!/bin/sh
# entrypoint.sh

set -e 
# Run Django database migrations
echo "Running migrations..."
python manage.py migrate

# Start the Django development server
echo "Starting server..."
exec python manage.py runserver 0.0.0.0:8000 --noreload
