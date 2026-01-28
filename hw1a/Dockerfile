# ---- build stage ----
FROM golang:1.25 AS builder      # Base image with Go compiler, named "builder"
WORKDIR /app                     # Set working directory

COPY go.mod go.sum ./            # Copy dependency files
RUN go mod download              # Download dependencies

COPY . .                         # Copy all source code
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .   # Compile static binary

# ---- run stage ----
FROM gcr.io/distroless/base-debian12   # Minimal runtime image (no shell, more secure)
WORKDIR /                              # Set root as working directory
COPY --from=builder /app/server /server   # Copy binary from builder stage
EXPOSE 8080                            # Declare port
USER nonroot:nonroot                   # Run as non-root user (security)
ENTRYPOINT ["/server"]                 # Command to run when container starts