package logs

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/logicmonitor/lm-data-sdk-go/utils"
)

// LMLogIngest configuration options
type Option func(*LMLogIngest) error

// WithLogBatchingInterval is used for configuring batching time interval for logs.
func WithLogBatchingInterval(batchingInterval time.Duration) Option {
	return func(logIngest *LMLogIngest) error {
		logIngest.batch.interval = batchingInterval
		return nil
	}
}

// WithLogBatchingDisabled is used for disabling log batching.
func WithLogBatchingDisabled() Option {
	return func(logIngest *LMLogIngest) error {
		logIngest.batch.enabled = false
		return nil
	}
}

// WithAuthentication is used for passing authentication token if not set in environment variables.
func WithAuthentication(authParams utils.AuthParams) Option {
	return func(logIngest *LMLogIngest) error {
		logIngest.auth = authParams
		return nil
	}
}

// WithGzipCompression can be used to enable/disable gzip compression of log payload
// Note: By default, gzip compression is enabled.
func WithGzipCompression(gzip bool) Option {
	return func(logIngest *LMLogIngest) error {
		logIngest.gzip = gzip
		return nil
	}
}

// WithRateLimit is used to limit the log request count per minute
func WithRateLimit(requestCount int) Option {
	return func(logIngest *LMLogIngest) error {
		logIngest.rateLimiterSetting.RequestCount = requestCount
		return nil
	}
}

// WithHTTPClient is used to set HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(logIngest *LMLogIngest) error {
		logIngest.client = client
		return nil
	}
}

// WithEndpoint is used to set Endpoint URL to export logs
func WithEndpoint(endpoint string) Option {
	return func(logIngest *LMLogIngest) error {
		logIngest.url = endpoint
		return nil
	}
}

// WithResourceMappingOperation is used to set the operation to be used for device mapping
func WithResourceMappingOperation(op string) Option {
	return func(logIngest *LMLogIngest) error {
		if !strings.EqualFold(op, ResourceMappingOp_AND) && !strings.EqualFold(op, ResourceMappingOp_OR) {
			return fmt.Errorf("invalid resource mapping operation: %s", op)
		}
		logIngest.resourceMappingOp = op
		return nil
	}
}

// WithUserAgent sets the provided user agent
func WithUserAgent(userAgent string) Option {
	return func(logIngest *LMLogIngest) error {
		logIngest.userAgent = userAgent
		return nil
	}
}

type SendLogsOptionalParameters struct {
}

func NewSendLogOptionalParameters() *SendLogsOptionalParameters {
	return &SendLogsOptionalParameters{}
}
