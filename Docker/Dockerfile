### Step 1: Build stage
FROM golang:1-alpine AS builder

WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code and build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o docker-pihole-customdns

### 
## Step 2: Runtime stage
FROM scratch

# Copy only the binary from the build stage to the final image
COPY --from=builder /app/docker-pihole-customdns /

# Set the entry point for the container
ENTRYPOINT ["/docker-pihole-customdns"]
