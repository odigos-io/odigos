import { TraceSignalGeneralConfig } from "./types";

export interface ResourceAttributeFromServer {
  key: string;
  value: string;
}

export interface OpAMPSdkConfiguration {
    remoteResourceAttributes: ResourceAttributeFromServer[];
    traceSignal: TraceSignalGeneralConfig;
}
