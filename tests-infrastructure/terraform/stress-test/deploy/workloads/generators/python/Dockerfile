# Build stage
FROM python:3.12-alpine AS builder

WORKDIR /app

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt --target /app/site-packages

# Copy source code
COPY . .

# Final stage
FROM python:3.12-alpine

WORKDIR /app

# Copy Python packages from builder
COPY --from=builder /app/site-packages /app/site-packages

# Copy application code
COPY app.py .

# Set Python path
ENV PYTHONPATH=/app/site-packages

# Run the application
CMD ["python", "app.py"]
