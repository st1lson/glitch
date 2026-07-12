# Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application statically with optimizations
# -s: Omit the symbol table and debug information
# -w: Omit the DWARF symbol table
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o glitch ./cmd/glitch

# Final Stage
FROM alpine:latest

# Install CA certificates for HTTPS proxying
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/glitch /usr/local/bin/glitch

# Expose the default port
EXPOSE 3000

# Command to run the executable
ENTRYPOINT ["glitch"]
