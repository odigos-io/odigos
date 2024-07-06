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

  onRemoteResource?: (remoteResource: Resource) => void;
  onNewInstrumentationLibrariesConfiguration?: (configs: InstrumentationLibraryConfiguration[]) => void;
}

export interface ResourceAttributeFromServer {
  key: string;
  value: string;
}

export interface InstrumentationLibraryConfiguration {
  name: string;
  enabled: boolean;
}
