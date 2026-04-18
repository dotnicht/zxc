#!/bin/bash
set -eu
CONF="$(dirname "$0")/script.conf"
URL=$(sed -n '1p' "$CONF")
TOKEN=$(sed -n '2p' "$CONF")
curl -s -X POST "$URL" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"event":"deployed"}'
