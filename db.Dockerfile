# syntax=docker/dockerfile:1

ARG PG_VERSION=16
ARG DEBIAN_CODENAME=bookworm
ARG MIGRATE_VERSION=v4.16.2

FROM postgres:${PG_VERSION}-${DEBIAN_CODENAME}

ARG MIGRATE_VERSION
ARG TARGETARCH

ADD https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-${TARGETARCH}.tar.gz \
	/migrate/migrate.tar.gz

RUN cd /migrate && \
	tar xvfz migrate.tar.gz

COPY migrations /migrate/migrations/
COPY test-db /docker-entrypoint-initdb.d/
