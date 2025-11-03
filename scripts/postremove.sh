#!/bin/sh
# Reload systemd daemon
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
fi
echo ""
echo "Allstar Nexus has been removed."
echo ""
echo "Configuration and data were preserved:"
echo "  /etc/allstar-nexus/config.yaml"
echo "  /var/lib/allstar-nexus"
echo ""
echo "Remove them manually if desired:"
echo "  sudo rm -rf /etc/allstar-nexus"
echo "  sudo rm -rf /var/lib/allstar-nexus"
echo ""
