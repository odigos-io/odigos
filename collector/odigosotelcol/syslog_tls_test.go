package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/config/testconnection"
)

func TestSyslogTLSAndRetry(t *testing.T) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Syslog test CA"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM([]byte(caPEM)))
	makeCert := func(serial int64, expired bool) (tls.Certificate, string, string) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		leaf := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: "localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}}
		if expired {
			leaf.NotAfter = time.Now().Add(-time.Minute)
		}
		der, err := x509.CreateCertificate(rand.Reader, leaf, ca, &key.PublicKey, caKey)
		require.NoError(t, err)
		keyDER, err := x509.MarshalECPrivateKey(key)
		require.NoError(t, err)
		certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
		keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
		cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
		require.NoError(t, err)
		return cert, certPEM, keyPEM
	}
	serverCert, _, _ := makeCert(2, false)
	_, clientCert, clientKey := makeCert(3, false)
	_, expiredCert, expiredKey := makeCert(4, true)
	for _, tc := range []struct {
		name                                                    string
		mtls, badCA, badHostname, missingClient, expired, retry bool
	}{
		{name: "TLS private CA"},
		{name: "mTLS", mtls: true},
		{name: "untrusted server", badCA: true},
		{name: "wrong hostname", badHostname: true},
		{name: "missing client certificate", mtls: true, missingClient: true},
		{name: "expired client certificate", mtls: true, expired: true},
		{name: "retry after receiver starts", mtls: true, retry: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			address := ln.Addr().String()
			_, port, _ := net.SplitHostPort(address)
			if tc.retry {
				require.NoError(t, ln.Close())
			}
			defer func() { ln.Close() }()
			data := map[string]string{"SYSLOG_ENDPOINT": "127.0.0.1", "SYSLOG_PORT": port, "SYSLOG_CA_PEM": caPEM}
			if tc.mtls {
				data["SYSLOG_MTLS_ENABLED"] = "true"
			}
			dest := &testconnection.TestConnectionConfig{ID: "tls-test", DestinationType: "syslog", Signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}, Config: data}
			cfg := &config.Config{Exporters: config.GenericMap{}, Processors: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
			_, err = (&config.Syslog{}).ModifyConfig(dest, cfg)
			require.NoError(t, err)
			values := cfg.Exporters["syslog/tls-test"].(config.GenericMap)
			tlsValues := values["tls"].(config.GenericMap)
			if tc.badCA {
				delete(tlsValues, "ca_pem")
			}
			if tc.badHostname {
				tlsValues["server_name_override"] = "wrong.example.com"
			}
			if tc.mtls {
				tlsValues["cert_pem"] = clientCert
				tlsValues["key_pem"] = clientKey
				if tc.expired {
					tlsValues["cert_pem"] = expiredCert
					tlsValues["key_pem"] = expiredKey
				}
				if tc.missingClient {
					delete(tlsValues, "cert_pem")
					delete(tlsValues, "key_pem")
				}
			}
			values["sending_queue"] = map[string]any{"enabled": false}
			values["retry_on_failure"] = config.GenericMap{"enabled": tc.retry, "initial_interval": "10ms", "max_interval": "50ms", "max_elapsed_time": "2s"}
			factories, err := components()
			require.NoError(t, err)
			factory := factories.Exporters[component.MustNewType("syslog")]
			exporterCfg := factory.CreateDefaultConfig()
			require.NoError(t, syslogTestConf(t, values).Unmarshal(exporterCfg))
			exp, err := factory.CreateLogs(t.Context(), exporter.Settings{ID: component.MustNewID("syslog"), TelemetrySettings: componenttest.NewNopTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, exporterCfg)
			require.NoError(t, err)
			require.NoError(t, exp.Start(t.Context(), componenttest.NewNopHost()))
			defer exp.Shutdown(context.Background())
			serverConfig := &tls.Config{Certificates: []tls.Certificate{serverCert}, MinVersion: tls.VersionTLS12, ClientCAs: pool}
			if tc.mtls {
				serverConfig.ClientAuth = tls.RequireAndVerifyClientCert
			}
			received := make(chan error, 1)
			startReceiver := func() {
				go func() {
					conn, err := ln.Accept()
					if err != nil {
						received <- err
						return
					}
					defer conn.Close()
					conn.SetDeadline(time.Now().Add(2 * time.Second))
					secure := tls.Server(conn, serverConfig)
					if err = secure.Handshake(); err != nil {
						received <- err
						return
					}
					payload, err := io.ReadAll(secure)
					if err == nil && len(payload) == 0 {
						err = io.ErrUnexpectedEOF
					}
					received <- err
				}()
			}
			if !tc.retry {
				startReceiver()
			}
			logs := plog.NewLogs()
			record := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			record.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			record.Attributes().PutStr("message", "TLS delivery")
			sent := make(chan error, 1)
			go func() { sent <- exp.ConsumeLogs(t.Context(), logs) }()
			if tc.retry {
				// Leave the port closed long enough to require at least one retry.
				select {
				case err := <-sent:
					t.Fatalf("export completed before recovery: %v", err)
				case <-time.After(100 * time.Millisecond):
				}
				ln, err = net.Listen("tcp", address)
				require.NoError(t, err)
				startReceiver()
			}
			rejected := tc.badCA || tc.badHostname || tc.missingClient || tc.expired
			select {
			case err := <-sent:
				if !rejected {
					require.NoError(t, err)
				}
			case <-time.After(4 * time.Second):
				t.Fatal("export timed out")
			}
			select {
			case err := <-received:
				if rejected {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("receiver timed out")
			}
		})
	}
	t.Run("malformed certificate", func(t *testing.T) {
		factories, err := components()
		require.NoError(t, err)
		factory := factories.Exporters[component.MustNewType("syslog")]
		cfg := factory.CreateDefaultConfig()
		require.NoError(t, confmap.NewFromStringMap(map[string]any{"endpoint": "127.0.0.1", "port": 6514, "tls": map[string]any{"ca_pem": "invalid PEM"}}).Unmarshal(cfg))
		_, err = factory.CreateLogs(t.Context(), exporter.Settings{ID: component.MustNewID("syslog"), TelemetrySettings: componenttest.NewNopTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}, cfg)
		require.Error(t, err)
	})
}
