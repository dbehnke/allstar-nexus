#!/bin/sh
# Reload systemd daemon
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
fi
# Set ownership on data directory
if [ -d /var/lib/allstar-nexus ]; then
  chown allstar-nexus:allstar-nexus /var/lib/allstar-nexus
fi
echo ""
echo "Allstar Nexus has been installed!"
echo ""
echo "Next steps:"
echo "  1. Edit the configuration: /etc/allstar-nexus/config.yaml"
echo "  2. Enable the service: sudo systemctl enable allstar-nexus"
echo "  3. Start the service: sudo systemctl start allstar-nexus"
echo "  4. Check status: sudo systemctl status allstar-nexus"
echo ""
