#!/usr/bin/env bash

# This script is run automatically by the DB container during its initialization.

set -eux

/migrate/migrate \
	-database "postgresql:///${POSTGRES_DB}?host=/var/run/postgresql&sslmode=disable&user=${POSTGRES_USER}&sslmode=disable" \
	-path /migrate/migrations \
	-verbose \
	up
