# ---- build stage ----
# Base image with Go compiler, named "builder"
FROM golang:1.25 AS builder
# Set working directory
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy all source code
COPY . .
# Compile static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .

# ---- run stage ----
# Minimal runtime image (no shell, more secure)
FROM gcr.io/distroless/base-debian12
# Set root as working directory
WORKDIR /
# Copy binary from builder stage
COPY --from=builder /app/server /server
# Declare port
EXPOSE 8080
# Run as non-root user (security)
USER nonroot:nonroot
# Command to run when container starts
ENTRYPOINT ["/server"]