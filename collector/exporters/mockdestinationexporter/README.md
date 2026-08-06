# Mock Destination Exporter

This exporter can be used for development and testing.
It allows you to mock a specific behavior of a destination exporter.

## Configuration

The following configuration options are available:

- `response_duration` can be used to set the duration of time until the export response is returned. can be used to simulate slow receivers (due to errors, network issues, etc).
- `reject_fraction` number from 0 to 1 that determines the fraction of exports that mocks a rejection of the export request.
- `encoding` one of `proto` (default), `json` or `none`. When set to `proto` or `json`, the exporter serializes the telemetry into the OTLP wire format and discards the result. This simulates the CPU a real destination spends encoding data, which scales with the payload size. Set to `none` to skip serialization entirely.

Example:

```yaml
│   mockdestination:
│     reject_fraction: 0.5
│     response_duration: 500ms
│     encoding: proto
```
