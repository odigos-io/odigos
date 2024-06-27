# Odigos Sampling processor

This processor samples traces based on the following supported rules:

1. HTTP Latency Rule: This rule allows you to configure service, endpoint, and threshold. Traces with a duration less than the specified threshold will be deleted.


``` yaml
  groupbytrace:
    wait_duration: 10s
  odigossampling:                                                                                                                                                                                         
    rules:
      - name: "http-latency-test"
        type: "http_latency"
        rule_details: 
          "threshold": 1050
          "endpoint": "/buy"
          "service": "frontend"
```
- threshold: The maximum allowable trace duration in milliseconds. Traces with a duration less than this value will be deleted.
- endpoint: The specific HTTP route prefix to match for sampling. Only traces with routes starting with this prefix will be considered. For example, configuring /buy will also match /buy/product.
- service: The name of the service for which the rule applies. Only traces from this service will be considered.


**Notes:**
- When using the `odigossampling` processor, it is mandatory to use the `groupbytrace` processor beforehand.
```
service:
  pipelines:
    traces:
      receivers:
      processors:
      - groupbytrace
      - odigossampling
      exporters:
```