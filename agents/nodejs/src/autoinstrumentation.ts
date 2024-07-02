import { getNodeAutoInstrumentations } from "@opentelemetry/auto-instrumentations-node";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-grpc";
import { NodeSDK } from "@opentelemetry/sdk-node";
import { OpAMPClientHttp } from "./opamp";
import {
  SEMRESATTRS_TELEMETRY_SDK_LANGUAGE,
  TELEMETRYSDKLANGUAGEVALUES_NODEJS,
  SEMRESATTRS_PROCESS_PID,
} from "@opentelemetry/semantic-conventions";
import {
  Resource,
  envDetectorSync,
  hostDetectorSync,
  processDetectorSync,
} from "@opentelemetry/resources";
import { DiagConsoleLogger, DiagLogLevel, diag } from "@opentelemetry/api";
import { VERSION } from "./version";

// For development, uncomment the following line to see debug logs
// diag.setLogger(new DiagConsoleLogger(), DiagLogLevel.INFO);

// not yet published in '@opentelemetry/semantic-conventions'
const SEMRESATTRS_TELEMETRY_DISTRO_NAME = "telemetry.distro.name";
const SEMRESATTRS_TELEMETRY_DISTRO_VERSION = "telemetry.distro.version";

const opampServerHost = process.env.ODIGOS_OPAMP_SERVER_HOST;
const instrumentationDeviceId = process.env.ODIGOS_INSTRUMENTATION_DEVICE_ID;

if (!opampServerHost || !instrumentationDeviceId) {
  diag.error(
    "Missing required environment variables ODIGOS_OPAMP_SERVER_HOST and ODIGOS_INSTRUMENTATION_DEVICE_ID"
  );
} else {
  const opampClient = new OpAMPClientHttp({
    instrumentationDeviceId: instrumentationDeviceId,
    opAMPServerHost: opampServerHost,
    agentDescriptionIdentifyingAttributes: {
      [SEMRESATTRS_TELEMETRY_SDK_LANGUAGE]: TELEMETRYSDKLANGUAGEVALUES_NODEJS,
      [SEMRESATTRS_TELEMETRY_DISTRO_VERSION]: VERSION,
      [SEMRESATTRS_PROCESS_PID]: process.pid,
    },
    agentDescriptionNonIdentifyingAttributes: {},
  });

  opampClient.start();

  const sdk = new NodeSDK({
    resourceDetectors: [
      // env detector reads resource attributes from the environment.
      // we don't populate it at the moment, but if the user set anything, this detector will pick it up
      envDetectorSync,
      // info about executable, runtime, command, etc
      processDetectorSync,
      // host name, and arch
      hostDetectorSync,
      // attributes from OpAMP server, k8s attributes, service name, etc
      opampClient,
    ],
    // record additional data about the odigos distro
    resource: new Resource({
      [SEMRESATTRS_TELEMETRY_DISTRO_NAME]: "odigos",
      [SEMRESATTRS_TELEMETRY_DISTRO_VERSION]: VERSION,
    }),
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
  });
  sdk.start();

  const shutdown = async () => {
    try {
      diag.info("Shutting down OpenTelemetry SDK and OpAMP client");
      await Promise.all([sdk.shutdown(), opampClient.shutdown()]);
    } catch (err) {
      diag.error("Error shutting down OpenTelemetry SDK and OpAMP client", err);
    }
  };

  process.on("SIGTERM", shutdown);
  process.on("SIGINT", shutdown);
  process.on("exit", shutdown);
}
