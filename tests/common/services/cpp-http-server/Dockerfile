# Use an Alpine base image for building the server
FROM alpine:latest AS builder

# Install required packages: g++, wget, and make
RUN apk add --no-cache g++ wget cmake

# Download and install cpp-httplib
RUN wget https://github.com/yhirose/cpp-httplib/archive/refs/tags/v0.10.7.tar.gz -O cpp-httplib.tar.gz \
    && tar -xzf cpp-httplib.tar.gz \
    && mkdir -p /usr/local/include \
    && cp cpp-httplib-0.10.7/httplib.h /usr/local/include/httplib.h \
    && rm -rf cpp-httplib.tar.gz cpp-httplib-0.10.7

WORKDIR /app

# Copy the server source code into the container
COPY server.cpp .

# Build the HTTP server
RUN g++ -std=c++11 -I/usr/local/include/httplib server.cpp -o server

# Use a minimal runtime base image
FROM alpine:latest

# Install required shared libraries for running the application
RUN apk add --no-cache libstdc++

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/server /app/server

# Expose the port the HTTP server will run on
EXPOSE 3000

# Run the HTTP server
CMD ["/app/server"]
