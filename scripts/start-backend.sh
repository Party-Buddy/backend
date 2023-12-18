#!/usr/bin/env bash

set -eu

function log() {
	echo -n "start-backend.sh: "
	echo "$@"
}

if [[ -v "START_DB_PASSWORD_FILE" ]]; then
	if [[ ! -r "${START_DB_PASSWORD_FILE}" ]]; then
		echo "File ${START_DB_PASSWORD_FILE} not found or not readable!"
		exit 1
	fi

	log "reading \$PARTY_BUDDY_DB_PASSWORD from ${START_DB_PASSWORD_FILE}..."
	export PARTY_BUDDY_DB_PASSWORD=$(<"${START_DB_PASSWORD_FILE}")
fi

log "starting app..."

exec /app/party-buddy "$@"
