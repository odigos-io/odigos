if command -v systemctl >/dev/null 2>&1; then
    systemctl enable odigos-otelcol.service
    systemctl start odigos-otelcol.service
fi
