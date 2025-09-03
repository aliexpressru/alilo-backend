# Build stage
FROM golang:1.23-alpine AS builder

# Install required packages
RUN apk add --no-cache --virtual .build-deps \
    git \
    make \
    protobuf \
    ca-certificates \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Copy source code
COPY . .

# Install dependencies and build
RUN make install-tools && \
    make generate && \
    go mod vendor && \
    go mod tidy && \
    make build && \
    apk del .build-deps

# Final stage - minimal runtime image
FROM alpine:latest

# Install only essential runtime dependencies and clean cache
RUN apk add --no-cache --virtual .runtime-deps \
    ca-certificates \
    wget \
    && rm -rf /var/cache/apk/* \
    && apk del .runtime-deps

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/alilo-backend .
COPY --from=builder /app/templateSimpleScript .

# Set proper permissions
RUN chown appuser:appgroup /app/alilo-backend && \
    chmod +x /app/alilo-backend

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 8084 3000

# Entrypoint
ENTRYPOINT ["./alilo-backend"]