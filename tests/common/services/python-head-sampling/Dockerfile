FROM python:3.11.13-alpine
WORKDIR /app
COPY ./requirements.txt /app
RUN pip install --no-cache-dir -r requirements.txt
COPY ./src /app
CMD ["python", "main.py"]
