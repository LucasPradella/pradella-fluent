#!/usr/bin/env bash
# OWASP ZAP baseline scan against the local API (quickstart V12).
# Prereqs: docker; API running on :8080 (go run ./cmd/api -migrate -seed).
set -euo pipefail

TARGET="${1:-http://host.docker.internal:8080/api/v1}"

echo "ZAP baseline scan → ${TARGET}"
docker run --rm --add-host=host.docker.internal:host-gateway \
  -v "$(pwd)":/zap/wrk:rw \
  ghcr.io/zaproxy/zaproxy:stable \
  zap-baseline.py \
    -t "${TARGET}" \
    -r zap-report.html \
    -I

echo "Report: ./zap-report.html"
