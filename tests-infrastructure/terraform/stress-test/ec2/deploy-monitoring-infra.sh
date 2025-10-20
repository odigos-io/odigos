#!/bin/bash
set -euxo pipefail
exec > /var/log/user-data.log 2>&1

yum install -y xfsprogs curl tar yum-utils

# ---------------- Disk Setup ----------------
for dev in /dev/nvme1n1 /dev/nvme2n1 /dev/nvme3n1; do
  for i in {1..120}; do
    [ -b "$dev" ] && break || sleep 2
  done
done

mkfs -t xfs /dev/nvme1n1
mkdir -p /mnt/prometheus-data
echo "/dev/nvme1n1 /mnt/prometheus-data xfs defaults,nofail 0 2" >> /etc/fstab
mount /mnt/prometheus-data

mkfs -t xfs /dev/nvme2n1
mkdir -p /mnt/grafana-data
echo "/dev/nvme2n1 /mnt/grafana-data xfs defaults,nofail 0 2" >> /etc/fstab
mount /mnt/grafana-data

mkfs -t xfs /dev/nvme3n1
mkdir -p /mnt/clickhouse-data
echo "/dev/nvme3n1 /mnt/clickhouse-data xfs defaults,nofail 0 2" >> /etc/fstab
mount /mnt/clickhouse-data

# ---------------- Prometheus ----------------
cd /tmp
curl -sSLo prometheus.tar.gz https://github.com/prometheus/prometheus/releases/download/v2.52.0/prometheus-2.52.0.linux-amd64.tar.gz
tar -xzf prometheus.tar.gz
install -m 0755 prometheus-2.52.0.linux-amd64/prometheus /usr/local/bin/prometheus

id prometheus >/dev/null 2>&1 || useradd --no-create-home --shell /sbin/nologin prometheus
mkdir -p /etc/prometheus /var/lib/prometheus
chown -R prometheus:prometheus /etc/prometheus /var/lib/prometheus /mnt/prometheus-data

cat >/etc/prometheus/prometheus.yml <<'EOF'
global:
  scrape_interval: 30s
scrape_configs: [] # receiver-only
EOF

cat >/etc/systemd/system/prometheus.service <<'EOF'
[Unit]
Description=Prometheus (remote-write receiver)
After=network-online.target

[Service]
User=prometheus
ExecStart=/usr/local/bin/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --web.enable-remote-write-receiver \
  --storage.tsdb.path=/mnt/prometheus-data \
  --storage.tsdb.retention.time=7d
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now prometheus

# ---------------- Grafana ----------------
cat >/etc/yum.repos.d/grafana.repo <<'EOF'
[grafana]
name=Grafana OSS
baseurl=https://packages.grafana.com/oss/rpm
repo_gpgcheck=1
enabled=1
gpgcheck=1
gpgkey=https://packages.grafana.com/gpg.key
EOF
yum install -y grafana

cat >/etc/grafana/grafana.ini <<'EOF'
[paths]
data = /mnt/grafana-data
[server]
http_addr = 127.0.0.1
http_port = 3000
EOF
chown grafana:grafana /etc/grafana/grafana.ini

mkdir -p /etc/grafana/provisioning/datasources
cat >/etc/grafana/provisioning/datasources/prometheus.yaml <<'EOF'
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://127.0.0.1:9090
    isDefault: true
    jsonData:
      httpMethod: POST
EOF

# Create dashboard provisioning directory
mkdir -p /etc/grafana/provisioning/dashboards
cat >/etc/grafana/provisioning/dashboards/dashboard.yaml <<'EOF'
apiVersion: 1
providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
EOF

chown -R grafana:grafana /etc/grafana/provisioning /mnt/grafana-data

# Create script to import dashboard 15760 (Kubernetes Pods View)
cat >/usr/local/bin/import-grafana-dashboard.sh <<'EOF'
#!/bin/bash
set -e

# Wait for Grafana to be ready (max 5 minutes)
echo "Waiting for Grafana to be ready..."
for i in {1..60}; do
  if curl -s http://127.0.0.1:3000/api/health > /dev/null 2>&1; then
    echo "Grafana is ready after $((i*5)) seconds"
    break
  fi
  echo "Attempt $i/60: Grafana not ready yet, waiting 5 seconds..."
  sleep 5
done

# Check if Grafana is actually ready
if ! curl -s http://127.0.0.1:3000/api/health > /dev/null 2>&1; then
  echo "ERROR: Grafana failed to start within 5 minutes"
  exit 1
fi

# Wait for Grafana to fully initialize
sleep 10

# Download dashboard 15760 JSON
curl -s -L "https://grafana.com/api/dashboards/15760/revisions/latest/download" \
  -o /etc/grafana/provisioning/dashboards/kubernetes-pods-view.json

# Fix permissions
chown grafana:grafana /etc/grafana/provisioning/dashboards/kubernetes-pods-view.json

# Force Grafana to reload dashboards
curl -s -X POST http://127.0.0.1:3000/api/admin/provisioning/dashboards/reload || true

EOF

chmod +x /usr/local/bin/import-grafana-dashboard.sh

systemctl enable --now grafana-server

# Import dashboard after Grafana starts
nohup /usr/local/bin/import-grafana-dashboard.sh > /var/log/grafana-dashboard-import.log 2>&1 &

# ---------------- ClickHouse ----------------
rpm --import https://packages.clickhouse.com/rpm/stable/repodata/repomd.xml.key
yum-config-manager --add-repo https://packages.clickhouse.com/rpm/clickhouse.repo
yum install -y clickhouse-server clickhouse-client

mkdir -p /mnt/clickhouse-data/{data,tmp,user_files,format_schemas,metadata,metadata_dropped,preprocessed_configs,flags,access}
chown -R clickhouse:clickhouse /mnt/clickhouse-data
chmod -R 750 /mnt/clickhouse-data
chmod 755 /mnt/clickhouse-data/format_schemas

mkdir -p /etc/clickhouse-server/config.d
cat >/etc/clickhouse-server/config.d/01-paths.xml <<'EOF'
<clickhouse>
  <path>/mnt/clickhouse-data/</path>
  <tmp_path>/mnt/clickhouse-data/tmp/</tmp_path>
  <user_files_path>/mnt/clickhouse-data/user_files/</user_files_path>
  <format_schema_path>/mnt/clickhouse-data/format_schemas/</format_schema_path>
</clickhouse>
EOF

cat >/etc/clickhouse-server/config.d/02-network.xml <<'EOF'
<clickhouse>
  <listen_host>0.0.0.0</listen_host>
  <tcp_port>9000</tcp_port>
  <http_port>8123</http_port>
  <interserver_http_port>9012</interserver_http_port>
</clickhouse>
EOF

chown root:root /etc/clickhouse-server/config.d/*.xml
chmod 644 /etc/clickhouse-server/config.d/*.xml
chmod 755 /etc/clickhouse-server/config.d
chown clickhouse:clickhouse /etc/clickhouse-server/config.d

systemctl daemon-reload
systemctl enable clickhouse-server
systemctl start clickhouse-server

# ---------------- K6  ----------------
cd /usr/local/bin
curl -sSL https://github.com/grafana/k6/releases/download/v0.51.0/k6-v0.51.0-linux-amd64.tar.gz | tar xz
mv k6-v0.51.0-linux-amd64/k6 /usr/bin/k6
chmod +x /usr/bin/k6
rm -rf k6-v0.51.0-linux-amd64

mkdir -p /opt/k6/tests
# Create simple dummy K6 load test script
cat >/opt/k6/tests/loadtest.js <<'EOF'
import http from 'k6/http';

export const options = {
  vus: 1,
  duration: '30s',
};

export default function () {
  http.get('http://localhost:3000');
}
EOF

echo "Monitoring infrastructure deployment completed!"
