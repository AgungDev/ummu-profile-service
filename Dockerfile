# syntax=docker/dockerfile:1
FROM golang:1.24.3-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
COPY --from=ummu-go-auth . /ummu-go-outh
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o profile-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o import-auth-profiles ./cmd/import-auth-profiles

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/profile-service ./profile-service
COPY --from=builder /app/import-auth-profiles ./import-auth-profiles
EXPOSE 9002
ENTRYPOINT ["./profile-service"]
