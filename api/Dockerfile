# Use the official Golang image as a base
FROM golang:1.23.3 AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum ./
# # Set GOPROXY to direct to avoid proxy timeouts
ENV GOPROXY=direct
# # Install git for go mod download
# RUN echo "nameserver 8.8.8.8" > /etc/resolv.conf && \
#     echo "nameserver 8.8.4.4" >> /etc/resolv.conf && \
RUN apt-get update && apt-get install -y git ca-certificates && \
    go mod download


RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Use a smaller image for the final container
FROM alpine:latest

# Install CA certificates for HTTPS
# RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Expose the port the API will run on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
