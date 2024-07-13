import { Attributes } from "@opentelemetry/api";
import { Resource } from "@opentelemetry/resources";
import { PackageStatus } from "./generated/opamp_pb";
import { PartialMessage } from "@bufbuild/protobuf";

export interface OpAMPClientHttpConfig {
  // instrumentedDeviceId, as allocated by the kubelet,
  // and injected into the pod as an environment variable named ODIGOS_INSTRUMENTATION_DEVICE_ID
  // This is the unique identifier for the device that is mounted to the container
  instrumentationDeviceId: string;
  opAMPServerHost: string; // the host + (optional) port of the OpAMP server to connect over http://
  pollingIntervalMs?: number;

  agentDescriptionIdentifyingAttributes?: Attributes;
  agentDescriptionNonIdentifyingAttributes?: Attributes;

  initialPackageStatues: PartialMessage<PackageStatus>[];

  onNewRemoteConfig: (remoteConfig: RemoteConfig) => void;
}

// Sdk Remote Configuration

export interface TraceSignalGeneralConfig {
  enabled: boolean; // if enabled is false, the pipeline is not configured to receive spans
  defaultEnabledValue: boolean;
}

export interface SdkConfiguration {
  remoteResource: Resource; // parse resource object
  traceSignal: TraceSignalGeneralConfig;
}

// InstrumentationLibrary Remote Configuration
export interface InstrumentationLibraryTracesConfiguration {
  // if the value is set, use it, otherwise use the default value from the trace signal in the sdk level
  enabled?: boolean;
}
export interface InstrumentationLibraryConfiguration {
  name: string;
  traces: InstrumentationLibraryTracesConfiguration;
}

// All remote config fields

export type RemoteConfig = {
  sdk: SdkConfiguration;
  instrumentationLibraries: InstrumentationLibraryConfiguration[];
};