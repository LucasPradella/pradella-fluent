# Stage 1: Build the React/Vite PWA
FROM node:20-alpine AS frontend-build
WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build the Go API binary
FROM golang:1.26-alpine AS backend-build
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o api ./cmd/api

# Stage 3: Minimal runtime image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend-build /build/api       ./api
COPY --from=backend-build /build/migrations ./migrations
COPY --from=frontend-build /build/dist      ./dist
EXPOSE 8080
# -migrate: apply pending SQL migrations on startup (idempotent)
# -seed:    insert lessons/placement bank when tables are empty (idempotent)
# -static-dir: serve the PWA from ./dist, API routes stay under /api/v1
CMD ["./api", "-migrate", "-seed", "-static-dir", "./dist"]
