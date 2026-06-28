FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /app/dist ./ui/dist/
RUN go build -o /opensp8c ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend-builder /opensp8c /usr/local/bin/opensp8c
EXPOSE 8080
ENTRYPOINT ["opensp8c"]
