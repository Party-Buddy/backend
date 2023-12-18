# syntax=docker/dockerfile:1

ARG GO_VERSION=1.21
ARG DEBIAN_CODENAME=bookworm

FROM golang:${GO_VERSION}-${DEBIAN_CODENAME} AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 make build

ARG DEBIAN_CODENAME
FROM debian:${DEBIAN_CODENAME}

WORKDIR /app

COPY --from=build /app/party-buddy /app/party-buddy
COPY scripts/start-backend.sh /app/

ENTRYPOINT ["./start-backend.sh"]
