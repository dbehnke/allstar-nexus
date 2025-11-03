#!/bin/sh
# Create allstar-nexus user if it doesn't exist
if ! id allstar-nexus >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /sbin/nologin allstar-nexus
fi
