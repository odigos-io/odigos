if command -v systemctl >/dev/null 2>&1; then
    systemctl disable --now odigos-otelcol-config.path 2>/dev/null || true
    systemctl stop odigos-otelcol.service
    systemctl disable odigos-otelcol.service
    systemctl disable --now odigos-otelcol-config-reload.service 2>/dev/null || true
fi
