# Stage 1: build the Vue frontend
FROM node:24-alpine AS frontend
WORKDIR /app/web
COPY web/package.json .
RUN npm install
COPY web/ .
RUN npm run build

# Stage 2: build the Go binary
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
# Copy built frontend into the path the Go binary expects
COPY --from=frontend /app/internal/api/static ./internal/api/static
RUN go build -o letrvu ./cmd/letrvu

# Stage 3: minimal runtime image
FROM alpine:3.23
# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates tzdata
COPY --from=backend /app/letrvu /usr/local/bin/letrvu
EXPOSE 8080
ENTRYPOINT ["letrvu"]
CMD ["-addr", ":8080"]
