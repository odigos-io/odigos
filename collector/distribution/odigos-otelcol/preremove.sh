if command -v systemctl >/dev/null 2>&1; then
    systemctl stop odigos-otelcol.service
    systemctl disable odigos-otelcol.service
fi
