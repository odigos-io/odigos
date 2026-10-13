package config

import (
	"cmp"
	"fmt"
	"net"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	syslogRFC5424 = "rfc5424"
	syslogRFC3164 = "rfc3164"
)

type Syslog struct{}

func (s *Syslog) DestType() common.DestinationType {
	return common.SyslogDestinationType
}

func (s *Syslog) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	if !isLoggingEnabled(dest) {
		return nil, fmt.Errorf("syslog supports logs only")
	}
	for _, signal := range dest.GetSignals() {
		if signal != common.LogsObservabilitySignal {
			return nil, fmt.Errorf("syslog does not support %s", signal)
		}
	}
	exporterConfig, processorConfig, err := SyslogDestinationConfig(dest)
	if err != nil {
		return nil, err
	}
	name := "syslog/" + dest.GetID()
	processorName := "transform/syslog-" + dest.GetID()
	cfg.Exporters[name] = exporterConfig
	cfg.Processors[processorName] = processorConfig
	pipelineName := "logs/syslog-" + dest.GetID()
	cfg.Service.Pipelines[pipelineName] = Pipeline{Processors: []string{processorName}, Exporters: []string{name}}
	return []string{pipelineName}, nil
}

// SyslogDestinationConfig builds the shared exporter and ordinary-log mapping for syslog destinations.
func SyslogDestinationConfig(dest ExporterConfigurer) (GenericMap, GenericMap, error) {
	data := dest.GetConfig()
	endpoint := strings.TrimSpace(data["SYSLOG_ENDPOINT"])
	if endpoint == "" || strings.ContainsAny(endpoint, "/ \t\r\n") ||
		(strings.Contains(endpoint, ":") && net.ParseIP(strings.Trim(endpoint, "[]")) == nil) {
		return nil, nil, fmt.Errorf("SYSLOG_ENDPOINT must be a hostname or IP address without a scheme or port")
	}
	// The upstream sender joins endpoint and port with a colon, so IPv6 needs brackets.
	if strings.Contains(endpoint, ":") && !strings.HasPrefix(endpoint, "[") {
		endpoint = "[" + endpoint + "]"
	}
	network := cmp.Or(data["SYSLOG_NETWORK"], "tcp")
	if network != "tcp" && network != "udp" {
		return nil, nil, fmt.Errorf("SYSLOG_NETWORK must be tcp or udp")
	}
	protocol := cmp.Or(data["SYSLOG_PROTOCOL"], syslogRFC5424)
	if protocol != syslogRFC5424 && protocol != syslogRFC3164 {
		return nil, nil, fmt.Errorf("SYSLOG_PROTOCOL must be rfc5424 or rfc3164")
	}
	tlsConfig, port, err := syslogTLSConfig(dest, network)
	if err != nil {
		return nil, nil, err
	}
	facility := 16
	if value := data["SYSLOG_FACILITY"]; value != "" {
		var err error
		facility, err = parseInt("SYSLOG_FACILITY", value)
		if err != nil {
			return nil, nil, err
		}
	}
	if facility < 0 || facility > 23 {
		return nil, nil, fmt.Errorf("SYSLOG_FACILITY must be between 0 and 23")
	}
	maxMessage := 8192
	upperLimit := 64000
	if protocol == syslogRFC3164 {
		maxMessage, upperLimit = 480, 480
	}
	if value := data["SYSLOG_MAX_MESSAGE_BYTES"]; value != "" {
		var err error
		maxMessage, err = parseInt("SYSLOG_MAX_MESSAGE_BYTES", value)
		if err != nil {
			return nil, nil, err
		}
	}
	if maxMessage < 1 || maxMessage > upperLimit {
		return nil, nil, fmt.Errorf("SYSLOG_MAX_MESSAGE_BYTES must be between 1 and %d for %s", upperLimit, protocol)
	}
	octetCounting := protocol == syslogRFC5424 && network == "tcp"
	if value := data["SYSLOG_OCTET_COUNTING"]; value != "" && value != "auto" {
		if value != "true" && value != "false" {
			return nil, nil, fmt.Errorf("SYSLOG_OCTET_COUNTING must be auto, true, or false")
		}
		octetCounting = value == "true"
	}
	if octetCounting && (protocol != syslogRFC5424 || network != "tcp") {
		return nil, nil, fmt.Errorf("octet counting requires TCP and RFC5424")
	}
	return GenericMap{
		"endpoint": endpoint, "port": port, "network": network, "protocol": protocol,
		"enable_octet_counting": octetCounting, "tls": tlsConfig,
		"timeout":          "10s",
		"retry_on_failure": GenericMap{"enabled": true, "initial_interval": "1s", "max_interval": "30s", "max_elapsed_time": "300s"},
		"sending_queue":    GenericMap{"queue_size": 1000, "num_consumers": 1},
	}, SyslogLogProcessor(facility, maxMessage), nil
}

// SyslogLogProcessor maps ordinary OTLP logs to the attributes consumed by the upstream exporter.
// It is shared with vendor destinations that use the same syslog wire format.
func SyslogLogProcessor(facility, maxMessageBytes int) GenericMap {
	statements := []string{
		`set(log.attributes["message"], log.body.string)`,
		`replace_pattern(log.attributes["message"], "\\r", "\\r")`,
		`replace_pattern(log.attributes["message"], "\\n", "\\n")`,
		fmt.Sprintf(`truncate_all(log.attributes, %d)`, maxMessageBytes),
		`set(log.time, log.observed_time) where log.time_unix_nano == 0`,
		`set(log.time, Now()) where log.time_unix_nano == 0`,
		`set(log.attributes["hostname"], "-")`,
		`set(log.attributes["hostname"], resource.attributes["service.instance.id"]) where resource.attributes["service.instance.id"] != nil`,
		`set(log.attributes["hostname"], resource.attributes["k8s.pod.name"]) where resource.attributes["k8s.pod.name"] != nil`,
		`set(log.attributes["hostname"], resource.attributes["host.name"]) where resource.attributes["host.name"] != nil`,
		`set(log.attributes["hostname"], String(log.attributes["hostname"]))`,
		`replace_pattern(log.attributes["hostname"], "[^!-~]", "_")`,
		`set(log.attributes["hostname"], Substring(log.attributes["hostname"], 0, 255)) where Len(log.attributes["hostname"]) > 255`,
		`set(log.attributes["hostname"], "-") where log.attributes["hostname"] == ""`,
		`set(log.attributes["appname"], "-")`,
		`set(log.attributes["appname"], String(resource.attributes["service.name"])) where resource.attributes["service.name"] != nil`,
		`replace_pattern(log.attributes["appname"], "[^!-~]", "_")`,
		`set(log.attributes["appname"], Substring(log.attributes["appname"], 0, 48)) where Len(log.attributes["appname"]) > 48`,
		`set(log.attributes["appname"], "-") where log.attributes["appname"] == ""`,
		`set(log.attributes["proc_id"], "-")`,
		`set(log.attributes["proc_id"], String(resource.attributes["process.pid"])) where resource.attributes["process.pid"] != nil`,
		`replace_pattern(log.attributes["proc_id"], "[^!-~]", "_")`,
		`set(log.attributes["proc_id"], Substring(log.attributes["proc_id"], 0, 128)) where Len(log.attributes["proc_id"]) > 128`,
		`set(log.attributes["proc_id"], "-") where log.attributes["proc_id"] == ""`,
		`set(log.attributes["version"], 1)`,
		`set(log.attributes["msg_id"], "-")`,
		// The upstream structured-data formatter does not escape all RFC5424 delimiters.
		`delete_key(log.attributes, "structured_data")`,
		fmt.Sprintf(`set(log.attributes["priority"], %d)`, facility*8+6),
		fmt.Sprintf(`set(log.attributes["priority"], %d) where log.severity_number >= 1 and log.severity_number <= 8`, facility*8+7),
		fmt.Sprintf(`set(log.attributes["priority"], %d) where log.severity_number >= 13 and log.severity_number <= 16`, facility*8+4),
		fmt.Sprintf(`set(log.attributes["priority"], %d) where log.severity_number >= 17 and log.severity_number <= 20`, facility*8+3),
		fmt.Sprintf(`set(log.attributes["priority"], %d) where log.severity_number >= 21`, facility*8+2),
	}
	return GenericMap{"error_mode": "propagate", "log_statements": []GenericMap{{"context": "log", "statements": statements}}}
}

func syslogTLSConfig(dest ExporterConfigurer, network string) (GenericMap, int, error) {
	data := dest.GetConfig()
	port := 6514
	tlsEnabled := data["SYSLOG_TLS_ENABLED"] != "false"
	mtlsEnabled := data["SYSLOG_MTLS_ENABLED"] == "true"
	if network == "udp" && (tlsEnabled || mtlsEnabled) {
		return nil, 0, fmt.Errorf("UDP syslog requires SYSLOG_TLS_ENABLED=false and does not support mTLS")
	}
	tlsEnabled = tlsEnabled || mtlsEnabled
	if !tlsEnabled {
		port = 514
	}
	if value := data["SYSLOG_PORT"]; value != "" {
		var err error
		port, err = parseInt("SYSLOG_PORT", value)
		if err != nil {
			return nil, 0, err
		}
	}
	if port < 1 || port > 65535 {
		return nil, 0, fmt.Errorf("SYSLOG_PORT must be between 1 and 65535")
	}
	tlsConfig := GenericMap{"insecure": !tlsEnabled}
	if tlsEnabled {
		tlsConfig["min_version"] = "1.2"
		if ca := data["SYSLOG_CA_PEM"]; ca != "" {
			tlsConfig["ca_pem"] = ca
		}
		if mtlsEnabled {
			tlsConfig["cert_pem"] = SecretEnvPlaceholder("SYSLOG_CLIENT_CERT_PEM", dest)
			tlsConfig["key_pem"] = SecretEnvPlaceholder("SYSLOG_CLIENT_KEY_PEM", dest)
		}
	}
	return tlsConfig, port, nil
}
