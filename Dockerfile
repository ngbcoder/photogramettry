# Stage 1: build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: build backend (OpenCV + GoCV)
FROM golang:1.24 AS backend
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config libopencv-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=1 GOOS=linux go build -tags opencv -o /out/photogen-web ./cmd/photogen-web

# Stage 3: runtime
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    libopencv-dev ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /out/photogen-web /usr/local/bin/photogen-web
COPY --from=frontend /app/frontend/dist ./frontend/dist
EXPOSE 8080
ENTRYPOINT ["photogen-web", "--port", "8080"]
