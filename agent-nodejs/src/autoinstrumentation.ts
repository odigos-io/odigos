import { getNodeAutoInstrumentations } from "@opentelemetry/auto-instrumentations-node";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-grpc";
import { NodeSDK } from "@opentelemetry/sdk-node";
import { OpAMPClientHttp } from "./opamp";
import {
  SEMRESATTRS_TELEMETRY_SDK_LANGUAGE,
  TELEMETRYSDKLANGUAGEVALUES_NODEJS,
  SEMRESATTRS_PROCESS_PID,
} from "@opentelemetry/semantic-conventions";
import { Resource, envDetectorSync, hostDetectorSync, processDetectorSync } from "@opentelemetry/resources";
import { diag } from "@opentelemetry/api";
import { VERSION } from "./version";

// not yet published in '@opentelemetry/semantic-conventions'
const SEMRESATTRS_TELEMETRY_DISTRO_NAME = 'telemetry.distro.name';
const SEMRESATTRS_TELEMETRY_DISTRO_VERSION = 'telemetry.distro.version';

const opampServerHost = process.env.ODIGOS_OPAMP_SERVER_HOST;
const instrumentationDeviceId = process.env.ODIGOS_INSTRUMENTATION_DEVICE_ID;

if (opampServerHost && instrumentationDeviceId) {
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
      envDetectorSync, // env detector reads resource attributes from the environment
      processDetectorSync, // info about executable, runtime, command, etc
      hostDetectorSync, // host name, arch, machine id, etc
      opampClient // attributes from OpAMP server, regarding k8s, service name, etc
    ],
    // record additional data about the odigos distro
    resource: new Resource({
      [SEMRESATTRS_TELEMETRY_DISTRO_NAME]: 'odigos',
      [SEMRESATTRS_TELEMETRY_DISTRO_VERSION]: VERSION,
    }),
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
  });
  sdk.start();

  const shutdown = async () => {
    try {
      diag.info('Shutting down OpenTelemetry SDK and OpAMP client');
      await Promise.all([sdk.shutdown(), opampClient.shutdown()]);
    } catch(err) {
      diag.error('Error shutting down OpenTelemetry SDK and OpAMP client', err);
    }
  }

  process.on('SIGTERM', shutdown);  
  process.on('SIGINT', shutdown);
  process.on('exit', shutdown);

} else {
  const sdk = new NodeSDK({
    autoDetectResources: true,
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
  });  
  sdk.start();
}


