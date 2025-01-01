# Odigos Sampling processor

This processor samples traces based on the following supported rules, grouping into these categories:

1. Endpoint Rules:

- HTTP Latency Rule: This rule allows you to configure service, endpoint, and threshold. Traces with a duration less than the specified threshold will be deleted.


``` yaml
  groupbytrace:
    wait_duration: 10s
  odigossampling:                                                                                                                                                                                         
    rules:
      endpoint_rules:  
        - name: "http-latency-test"
          type: "http_latency"
          rule_details: 
            "threshold": 1050
            "http_route": "/buy"
            "service_name": "frontend"
            "fallback_sampling_ratio": 20.0
  ```
- threshold: The maximum allowable trace duration in milliseconds. Traces with a duration less than this value will be deleted.
- endpoint: The specific HTTP route prefix to match for sampling. Only traces with routes starting with this prefix will be considered. For example, configuring /buy will also match /buy/product.
- service: The name of the service for which the rule applies. Only traces from this service will be considered.
- fallback_sampling_ratio: specifies the percentage of traces that meet the service/http_route filter but fall below the threshold that you still want to retain. For example, if a rule is set for service A and http_route B with a minimum latency threshold of 1 second, you might still want to keep some traces below this threshold. Setting the ratio to 20% ensures that 20% of these traces will be retained.

2. Global Rules:
-  Error Rule: This rule allows you to configure a list of status codes [ERROR/OK/UNSET]. traces with a status code that not configured will be delete.

``` yaml
rules: 
  global_rules:
    - name: "error-rule"
      type: error
      rule_details:
        fallback_sampling_ratio: 50
```
- fallback_sampling_ratio: This parameter specifies the percentage of non-error traces you want to retain. For instance, setting it to 50 means you will see 100% of error traces and 50% of non-error traces.


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