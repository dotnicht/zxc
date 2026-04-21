#!/bin/bash
set -eu
CONF="$(dirname "$0")/script.conf"
URL=$(sed -n '1p' "$CONF")
TOKEN=$(sed -n '2p' "$CONF")
NODE_NAME="node-$(head -c 8 /dev/urandom | od -An -tx1 | tr -d ' \n')"

send_webhook() {
  curl -s -X POST "$URL" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"event\":\"deployed\",\"node_name\":\"$NODE_NAME\"}"
}

if [ "${1:-}" != "--run-loop" ]; then
  nohup bash "$0" --run-loop >/tmp/zxc-webhook-loop.log 2>&1 &
  exit 0
fi

send_webhook
while true; do
  sleep 20
  send_webhook
done
