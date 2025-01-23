# LogicMonitor Go Data SDK

LogicMonitor Go Data SDK is suitable for ingesting the metrics and logs into the LogicMonitor Platform.

## Overview
LogicMonitor's Push Metrics feature allows you to send metrics directly to the LogicMonitor platform via a dedicated API, removing the need to route the data through a LogicMonitor Collector. Once ingested, these metrics are presented alongside all other metrics gathered via LogicMonitor, providing a single pane of glass for metric monitoring and alerting.

Similarly, If a log integration isn’t available or you have custom logs that you want to analyze, you can send the logs directly to your LogicMonitor account via the logs ingestion API.

## Getting Started

### Installation

To use the LogicMonitor Go Data SDK in your Go module, you can simply run the following command:

```bash
go get -u github.com/logicmonitor/lm-data-sdk-go
```

### Authentication
While using LMv1 authentication set LOGICMONITOR_ACCESS_ID and LOGICMONITOR_ACCESS_KEY properties.
In case of BearerToken authentication set LOGICMONITOR_BEARER_TOKEN property. 
Company's name or Account name must be passed to LOGICMONITOR_ACCOUNT property. 
All properties can be set using environment variable.

| Environment variable |	Description |
| -------------------- |:--------------:|
|   LOGICMONITOR_ACCOUNT         |	Account name (Company Name) is your organization name |
|   LOGICMONITOR_ACCESS_ID       |	Access id while using LMv1 authentication.|
|   LOGICMONITOR_ACCESS_KEY      |	Access key while using LMv1 authentication.|
|   LOGICMONITOR_BEARER_TOKEN    |	BearerToken while using Bearer authentication.|

## Usage

### Metrics Ingestion

This is how you can initialise metrics client:

```go
import (
	"context"
	"fmt"
	"os"

	"github.com/logicmonitor/lm-data-sdk-go/api/metrics"
)

func main() {
	options := []metrics.Option{
		metrics.WithMetricBatchingInterval(3 * time.Second),
        ...
	}

	lmMetric, err := metrics.NewLMMetricIngest(context.Background(), options...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when initializing metric client: %v\n", err)
		return
	}
    ...
}
```

Here is the complete [example](https://github.com/logicmonitor/lm-data-sdk-go/blob/main/example/metrics/metricsingestion.go) for metrics ingestion.


#### Options

Following options can be used to create the metrics api client.

|   Option  |	Description |
| -------------------- |:----------------------------------:|
| `WithMetricBatchingInterval(batchinterval time.Duration)`  | Sets time interval to wait before performing next batching of metrics. Default value is `10s`. |
| `WithMetricBatchingDisabled()` | Disables batching of metrics. Default value is `Enabled`. |
| `WithGzipCompression(gzip bool)` | Enables / disables gzip compression of metric payload. Default value is `Enabled`. |
| `WithRateLimit(requestCount int)` | Sets limit on the number of requests to metrics API per minute. Default value is `100`. |
| `WithHTTPClient(client *http.Client)` | Sets custom HTTP Client. Default http client is configured with timeout of `5s`.|
| `WithEndpoint(endpoint string)` | Sets endpoint to send the metrics to. Default value is `https://${LOGICMONITOR_ACCOUNT}.logicmonitor.com/rest/`.|
| `WithAuthentication(authParams utils.AuthParams)`  | Sets authentication parameters. |

### Logs Ingestion

This is how you can initialise logs client:

```go
  import (
	"context"
	"fmt"
	"os"

	"github.com/logicmonitor/lm-data-sdk-go/api/logs"
)

func main() {
	options := []logs.Option{
		logs.WithLogBatchingDisabled(),
        ...
	}

	lmLog, err := logs.NewLMLogIngest(context.Background(), options...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when initializing log client: %v", err)
		return
	}
    ...
}
```

Here is the complete [example](https://github.com/logicmonitor/lm-data-sdk-go/blob/main/example/logs/logsingestion.go) for logs ingestion.


#### Options

Following options can be used to create the logs api client.


|   Option  |	Description |
| -------------------- |:---------------------------------------------:| 
| `WithLogBatchingInterval(batchinterval time.Duration)`  | Sets time interval to wait before performing next batching of logs. Default value is `10s`. |
| `WithLogBatchingDisabled()` | Disables batching of logs. Default value is `Enabled`. |
| `WithGzipCompression(gzip bool)` | Enables / disables gzip compression of logs payload. Default value is `Enabled`. |
| `WithRateLimit(requestCount int)` | Sets limit on the number of requests to logs API per minute. Default value is `100`. |
| `WithHTTPClient(client *http.Client)` | Sets custom HTTP Client. Default http client is configured with timeout of `5s`.|
| `WithEndpoint(endpoint string)` | Sets endpoint to send the logs to. Default value is `https://${LOGICMONITOR_ACCOUNT}.logicmonitor.com/rest/`|
| `WithResourceMappingOperation(op string)` | Sets resource mapping operation. Valid operations are `AND` & `OR`. |
| `WithUserAgent(userAgent string)`         | Sets user agent. |
| `WithAuthentication(authParams utils.AuthParams)`         | Sets authentication parameters. |


## License

Copyright, 2023, LogicMonitor, Inc.

This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0. If a copy of the MPL was not distributed with this file, You can obtain one at https://mozilla.org/MPL/2.0/.
