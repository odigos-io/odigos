import { getNodeAutoInstrumentations } from "@opentelemetry/auto-instrumentations-node";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-grpc";
import { NodeSDK } from "@opentelemetry/sdk-node";
import { OpAMPClientHttp } from "./opamp";
import {
  SEMRESATTRS_TELEMETRY_SDK_LANGUAGE,
  TELEMETRYSDKLANGUAGEVALUES_NODEJS,
  SEMRESATTRS_TELEMETRY_SDK_NAME,
  SEMRESATTRS_TELEMETRY_SDK_VERSION,
  SEMRESATTRS_PROCESS_PID,
} from "@opentelemetry/semantic-conventions";

const opampServerHost = process.env.ODIGOS_OPAMP_SERVER_HOST;

if (opampServerHost) {
  const opampClient = new OpAMPClientHttp({
    opAMPServerHost: opampServerHost,
    agentDescriptionIdentifyingAttributes: {
      [SEMRESATTRS_TELEMETRY_SDK_LANGUAGE]: TELEMETRYSDKLANGUAGEVALUES_NODEJS,
      [SEMRESATTRS_TELEMETRY_SDK_NAME]: "odigos",
      [SEMRESATTRS_TELEMETRY_SDK_VERSION]: "0.0.1",
      [SEMRESATTRS_PROCESS_PID]: process.pid,
    },
    agentDescriptionNonIdentifyingAttributes: {},
  });

  opampClient.start();
}

const sdk = new NodeSDK({
  autoDetectResources: true,
  instrumentations: [getNodeAutoInstrumentations()],
  traceExporter: new OTLPTraceExporter(),
});

sdk.start();
