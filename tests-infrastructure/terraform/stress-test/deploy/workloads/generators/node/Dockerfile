# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies
RUN npm install --omit=dev

# Copy source code
COPY . .

# Final stage
FROM node:20-alpine

WORKDIR /app

# Copy node_modules from builder
COPY --from=builder /app/node_modules ./node_modules

# Copy application code
COPY app.js .

# Set environment variables
ENV OTEL_SERVICE_NAME=node-span-generator
ENV OTEL_RESOURCE_ATTRIBUTES=service.name=node-span-generator,service.version=v1.0.0
ENV SPANS_PER_SEC=6000
ENV SPAN_BYTES=4000

# Run the application
CMD ["node", "app.js"]
