# Mock Destination Exporter

This exporter can be used for development and testing.
It allows you to mock a specific behavior of a destination exporter.

## Configuration

The following configuration options are available:

- `response_duration` can be used to set the duration of time until the export response is returned. can be used to simulate slow receivers (due to errors, network issues, etc).
- `reject_fraction` number from 0 to 1 that determines the fraction of exports that mocks a rejection of the export request.

Example:

```yaml
│   mockdestination:
│     reject_fraction: 0.5
│     response_duration: 500ms
```
