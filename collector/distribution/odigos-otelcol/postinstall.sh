if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload

    systemctl enable odigos-otelcol.service
    systemctl start odigos-otelcol.service

    systemctl enable odigos-otelcol-config.path
    systemctl start odigos-otelcol-config.path

    systemctl reload odigos-otelcol.service
fi
