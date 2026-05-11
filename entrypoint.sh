#!/bin/sh
set -e

if [ "${RUN_MIGRATION:-false}" = "true" ] || [ "${RUN_MIGRATION:-0}" = "1" ]; then
  /app/location-service migrate
fi

if [ "${RUN_IMPORT:-false}" = "true" ] || [ "${RUN_IMPORT:-0}" = "1" ]; then
  /app/location-service import -file "${IMPORT_FILE:-/app/data/wilayah.sql}"
fi

exec /app/location-service "$@"
