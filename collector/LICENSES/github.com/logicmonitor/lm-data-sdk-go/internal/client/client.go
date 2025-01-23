package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	rateLimiter "github.com/logicmonitor/lm-data-sdk-go/pkg/ratelimiter"
	"github.com/logicmonitor/lm-data-sdk-go/utils"
)

type RequestConfig struct {
	Client          *http.Client
	RateLimiter     rateLimiter.RateLimiter
	Url             string
	Body            []byte
	Uri             string
	Method          string
	Token           string
	Gzip            bool
	Headers         map[string]string
	PayloadMetadata interface{}
	UserAgent       string
}

func New() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: false, MinVersion: tls.VersionTLS12}
	clientTransport := (http.RoundTripper)(transport)
	return &http.Client{Transport: clientTransport, Timeout: 5 * time.Second}
}

// DoRequest compresses the payload and exports it to LM Platform
func DoRequest(ctx context.Context, reqConfig RequestConfig) (*http.Response, error) {
	if reqConfig.Token == "" {
		return nil, fmt.Errorf("missing authentication token")
	}
	payloadBody := reqConfig.Body
	var err error
	if reqConfig.Gzip {
		payloadBody, err = utils.Gzip(payloadBody)
		if err != nil {
			return nil, fmt.Errorf("error while compressing body: %v", err)
		}
	}
	reqBody := bytes.NewBuffer(payloadBody)
	fullURL := reqConfig.Url + reqConfig.Uri

	req, err := http.NewRequest(reqConfig.Method, fullURL, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", reqConfig.Token)
	req.Header.Add("User-Agent", reqConfig.UserAgent)

	if reqConfig.Gzip {
		req.Header.Add("Content-Encoding", "gzip")
	}

	for key, value := range reqConfig.Headers {
		req.Header.Set(key, value)
	}
	if acquire, err := reqConfig.RateLimiter.Acquire(reqConfig.PayloadMetadata); !acquire {
		return nil, err
	}

	httpResp, err := reqConfig.Client.Do(req)
	if err != nil {
		return nil, err
	}
	return httpResp, nil
}
