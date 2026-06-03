package otlpproxygrpcexporter

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
)

// connectProxyDialer returns a grpc.WithContextDialer-compatible dialer that
// reaches target through an HTTP CONNECT proxy. The dialer opens a TCP
// connection to the proxy, issues `CONNECT target`, and on a 200 hands the raw
// tunneled connection back to gRPC (which then negotiates TLS/HTTP2 over it
// end-to-end — the proxy never sees the payload). This is the gRPC equivalent
// of confighttp's proxy_url for HTTP exporters.
//
// proxyURL is assumed validated by Config.Validate (scheme + host present).
func connectProxyDialer(proxyURL string) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, target string) (net.Conn, error) {
		pu, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse proxy_url: %w", err)
		}

		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", pu.Host)
		if err != nil {
			return nil, fmt.Errorf("dial proxy %s: %w", pu.Host, err)
		}

		req := &http.Request{
			Method: http.MethodConnect,
			URL:    &url.URL{Opaque: target},
			Host:   target,
			Header: make(http.Header),
		}
		if pu.User != nil {
			if pw, ok := pu.User.Password(); ok {
				cred := base64.StdEncoding.EncodeToString([]byte(pu.User.Username() + ":" + pw))
				req.Header.Set("Proxy-Authorization", "Basic "+cred)
			}
		}
		if err := req.Write(conn); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("write CONNECT to %s: %w", pu.Host, err)
		}

		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, req)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("read CONNECT response from %s: %w", pu.Host, err)
		}
		if resp.StatusCode != http.StatusOK {
			_ = conn.Close()
			return nil, fmt.Errorf("proxy %s CONNECT to %s failed: %s", pu.Host, target, resp.Status)
		}
		// If the proxy sent bytes past the headers, preserve them.
		if br.Buffered() > 0 {
			return &bufferedConn{Conn: conn, r: br}, nil
		}
		return conn, nil
	}
}

// bufferedConn wraps a net.Conn whose reader has bytes buffered past the CONNECT
// response, so those bytes are not lost when gRPC starts reading.
type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (b *bufferedConn) Read(p []byte) (int, error) { return b.r.Read(p) }
