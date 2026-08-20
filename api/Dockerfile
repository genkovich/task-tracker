# Stage 1: Build
FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /bin/task-tracker-api ./cmd/api
RUN CGO_ENABLED=0 go build -o /bin/task-tracker-migrate ./cmd/migrate

# Stage 2: Runtime
FROM alpine:3.24
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -s /sbin/nologin appuser
COPY --from=builder /bin/task-tracker-api /bin/task-tracker-api
COPY --from=builder /bin/task-tracker-migrate /bin/task-tracker-migrate
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD ["wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/livez"]
CMD ["/bin/task-tracker-api"]
