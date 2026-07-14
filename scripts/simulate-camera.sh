#!/usr/bin/env bash
# Simulates a camera mapper: periodically patches the camera-01 DeviceStatus
# with fake readings, standing in for what a real KubeEdge Mapper would
# report via the DMI gRPC interface.
set -euo pipefail

NAMESPACE="kubeedge"
DEVICE="camera-01"
INTERVAL="${1:-10}"
FRAME_COUNT=0

echo "Simulando camara '$DEVICE' cada ${INTERVAL}s (Ctrl+C para detener)..."

while true; do
  FRAME_COUNT=$((FRAME_COUNT + 1))
  TS=$(($(date +%s%N) / 1000000))

  # ~20% chance of detecting motion, with a plausible confidence score
  if (( RANDOM % 5 == 0 )); then
    MOTION="true"
    CONFIDENCE=$(awk -v seed="$RANDOM" 'BEGIN{srand(seed); printf "%.2f", 0.55 + rand()*0.44}')
  else
    MOTION="false"
    CONFIDENCE="0.0"
  fi

  # A transient API-server timeout must not kill the loop — a real mapper
  # keeps sampling while offline and reports again when the link is back.
  if kubectl patch devicestatus "$DEVICE" -n "$NAMESPACE" --type=merge --request-timeout=15s -p "{
    \"status\": {
      \"state\": \"Online\",
      \"lastOnlineTime\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
      \"twins\": [
        {\"propertyName\": \"status\", \"reported\": {\"value\": \"online\", \"metadata\": {\"timestamp\": \"$TS\"}}},
        {\"propertyName\": \"motionDetected\", \"reported\": {\"value\": \"$MOTION\", \"metadata\": {\"timestamp\": \"$TS\"}}},
        {\"propertyName\": \"frameCount\", \"reported\": {\"value\": \"$FRAME_COUNT\", \"metadata\": {\"timestamp\": \"$TS\"}}},
        {\"propertyName\": \"confidence\", \"reported\": {\"value\": \"$CONFIDENCE\", \"metadata\": {\"timestamp\": \"$TS\"}}}
      ]
    }
  }" > /dev/null 2>&1; then
    echo "$(date +%T) frame=$FRAME_COUNT motion=$MOTION confidence=$CONFIDENCE"
  else
    echo "$(date +%T) frame=$FRAME_COUNT PATCH FAILED (transient?) — reintento en el proximo ciclo"
  fi
  sleep "$INTERVAL"
done
