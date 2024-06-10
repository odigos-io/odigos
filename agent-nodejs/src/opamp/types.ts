import { Attributes } from "@opentelemetry/api";

export interface OpAMPClientHttpConfig {
  // instrumentedDeviceId, as allocated by the kubelet,
  // and injected into the pod as an environment variable named ODIGOS_INSTRUMENTATION_DEVICE_ID
  // This is the unique identifier for the device that is mounted to the container
  instrumentationDeviceId: string;
  opAMPServerHost: string; // the host + (optional) port of the OpAMP server to connect over http://
  pollingInterval?: number;

  agentDescriptionIdentifyingAttributes?: Attributes;
  agentDescriptionNonIdentifyingAttributes?: Attributes;
}

export interface ResourceAttributeFromServer {
  key: string;
  value: string;
}
