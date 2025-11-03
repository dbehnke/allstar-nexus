#!/bin/sh
# Stop service if running
if command -v systemctl >/dev/null 2>&1; then
  systemctl stop allstar-nexus.service 2>/dev/null || true
  systemctl disable allstar-nexus.service 2>/dev/null || true
fi
