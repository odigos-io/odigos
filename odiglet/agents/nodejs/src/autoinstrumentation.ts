import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';

import { NodeSDK } from '@opentelemetry/sdk-node';

const sdk = new NodeSDK({
    autoDetectResources: true,
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
});

sdk.start();