# Use Node.js 18 Alpine image
FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package.json .

# Install dependencies
RUN npm install

# Copy source code
COPY app.js .

# Set environment variables
ENV TOTAL_SPANS=10000
ENV SPAN_BYTES=2000

# Run the application
CMD ["npm", "start"]
