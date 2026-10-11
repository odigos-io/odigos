# Use official Node.js runtime as base image (pinned to exact version)
FROM node:20.19.2-alpine

# Set working directory in container
WORKDIR /app

# Copy package files
COPY package.json ./

# Install dependencies (even though we don't have any, good practice)
RUN npm install --production

# Copy application code
COPY index.js ./

# Expose port 3000
EXPOSE 3000

# Create non-root user for security
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nodejs -u 1001

# Change ownership of app directory
RUN chown -R nodejs:nodejs /app
USER nodejs

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

# Start the application
CMD ["npm", "start"]
