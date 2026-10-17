package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/leodido/go-syslog/v4/rfc3164"
	"github.com/leodido/go-syslog/v4/rfc5424"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/config/testconnection"
)

func TestSyslogDestinationWireFormats(t *testing.T) {
	for _, tc := range []struct {
		protocol string
		octets   bool
	}{{"rfc5424", true}, {"rfc5424", false}, {"rfc3164", false}} {
		t.Run(fmt.Sprintf("%s/octets=%t", tc.protocol, tc.octets), func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()
			_, port, _ := net.SplitHostPort(listener.Addr().String())
			dest := &testconnection.TestConnectionConfig{ID: "syslog-test", DestinationType: "syslog", Signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}, Config: map[string]string{
				"SYSLOG_ENDPOINT": "127.0.0.1", "SYSLOG_PORT": port, "SYSLOG_TLS_ENABLED": "false", "SYSLOG_PROTOCOL": tc.protocol, "SYSLOG_OCTET_COUNTING": strconv.FormatBool(tc.octets), "SYSLOG_MAX_MESSAGE_BYTES": "128",
			}}
			cfg := &config.Config{Exporters: config.GenericMap{}, Processors: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
			configers, err := config.LoadConfigers()
			require.NoError(t, err)
			_, err = configers[common.SyslogDestinationType].ModifyConfig(dest, cfg)
			require.NoError(t, err)
			factories, err := components()
			require.NoError(t, err)
			factory := factories.Exporters[component.MustNewType("syslog")]
			exporterCfg := factory.CreateDefaultConfig()
			expMap := cfg.Exporters["syslog/syslog-test"].(config.GenericMap)
			// Synchronous export surfaces delivery failures directly to this test.
			expMap["sending_queue"] = map[string]any{"enabled": false}
			require.NoError(t, syslogTestConf(t, expMap).Unmarshal(exporterCfg))
			exp, err := factory.CreateLogs(t.Context(), exporter.Settings{ID: component.MustNewID("syslog"), TelemetrySettings: componenttest.NewNopTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, exporterCfg)
			require.NoError(t, err)
			require.NoError(t, exp.Start(t.Context(), componenttest.NewNopHost()))
			defer exp.Shutdown(context.Background())
			transformFactory := factories.Processors[component.MustNewType("transform")]
			processorCfg := transformFactory.CreateDefaultConfig()
			require.NoError(t, syslogTestConf(t, cfg.Processors["transform/syslog-syslog-test"]).Unmarshal(processorCfg))
			sink := &consumertest.LogsSink{}
			proc, err := transformFactory.CreateLogs(t.Context(), processor.Settings{ID: component.MustNewID("transform"), TelemetrySettings: componenttest.NewNopTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, processorCfg, sink)
			require.NoError(t, err)
			require.NoError(t, proc.Start(t.Context(), componenttest.NewNopHost()))
			defer proc.Shutdown(context.Background())
			logs := plog.NewLogs()
			now := pcommon.NewTimestampFromTime(time.Now().UTC())
			for i, body := range []string{"ordinary log\nsecond line\rreturn", strings.Repeat("😀", 100)} {
				resource := logs.ResourceLogs().AppendEmpty()
				resource.Resource().Attributes().PutStr("service.name", fmt.Sprintf("service-%d", i))
				resource.Resource().Attributes().PutStr("host.name", fmt.Sprintf("host-%d", i))
				resource.Resource().Attributes().PutInt("process.pid", 42)
				record := resource.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.SetObservedTimestamp(now)
				record.SetSeverityNumber(plog.SeverityNumberError)
				record.Body().SetStr(body)
				record.Attributes().PutStr("structured_data", "invalid ] injection")
			}
			require.NoError(t, proc.ConsumeLogs(t.Context(), logs))
			transformed := sink.AllLogs()[0]
			require.Equal(t, now, transformed.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Timestamp())
			payload := make(chan []byte, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				b, _ := io.ReadAll(conn)
				payload <- b
			}()
			require.NoError(t, exp.ConsumeLogs(t.Context(), transformed))
			var raw []byte
			select {
			case raw = <-payload:
			case <-time.After(6 * time.Second):
				t.Fatal("no syslog payload")
			}
			reader := bufio.NewReader(strings.NewReader(string(raw)))
			for i := 0; i < 2; i++ {
				var frame []byte
				if tc.octets {
					prefix, err := reader.ReadString(' ')
					require.NoError(t, err)
					n, err := strconv.Atoi(strings.TrimSpace(prefix))
					require.NoError(t, err)
					frame = make([]byte, n)
					_, err = io.ReadFull(reader, frame)
					require.NoError(t, err)
				} else {
					frame, err = reader.ReadBytes('\n')
					require.NoError(t, err)
				}
				frame = []byte(strings.TrimSuffix(string(frame), "\n"))
				require.True(t, utf8.Valid(frame))
				if tc.protocol == "rfc5424" {
					msg, err := rfc5424.NewParser().Parse(frame)
					require.NoError(t, err, string(frame))
					parsed := msg.(*rfc5424.SyslogMessage)
					require.Equal(t, fmt.Sprintf("service-%d", i), *parsed.Appname)
					require.Equal(t, fmt.Sprintf("host-%d", i), *parsed.Hostname)
					require.Equal(t, uint8(3), *parsed.Severity)
					require.Equal(t, "42", *parsed.ProcID)
					if i == 0 {
						require.Equal(t, `ordinary log\nsecond line\rreturn`, *parsed.Message)
					} else {
						require.Len(t, *parsed.Message, 128)
					}
				} else {
					_, err := rfc3164.NewParser().Parse(frame)
					require.NoError(t, err, string(frame))
					require.Contains(t, string(frame), fmt.Sprintf("host-%d service-%d:", i, i))
				}
			}
			_, err = reader.ReadByte()
			require.ErrorIs(t, err, io.EOF, "multiline body must not create extra frames")
		})
	}
}

func syslogTestConf(t *testing.T, value any) *confmap.Conf {
	t.Helper()
	// Collector configuration is serialized before loading; normalize named maps and slices the same way.
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	var data map[string]any
	require.NoError(t, json.Unmarshal(raw, &data))
	return confmap.NewFromStringMap(data)
}
