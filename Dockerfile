# Build stage
FROM golang:1.23-alpine AS builder

# Install required packages
RUN apk add --no-cache git make protobuf ca-certificates

WORKDIR /app

# Copy source code
COPY . .

# Install dependencies and build
RUN make install-tools
RUN make generate
RUN go mod vendor
RUN go mod tidy
RUN make build

# Final stage
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/alilo-backend .

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080 8084 3000

# Run the application
CMD ["./alilo-backend"]